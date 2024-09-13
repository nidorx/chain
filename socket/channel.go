package socket

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/pkg"
	"github.com/nidorx/chain/pubsub"
)

// LeaveReason reasons why a LeaveHandler is invoked
type LeaveReason int

const (
	LeaveReasonLeave  = LeaveReason(0) // Client called _leave event (channel.leave()).
	LeaveReasonRejoin = LeaveReason(1) // Client called _join and there is already an active socket for the same topic
	LeaveReasonClose  = LeaveReason(2) // Connection lost and session is terminated. See Session.ScheduleShutdown
)

var (
	ErrJoinCrashed    = fmt.Errorf("join crashed")
	ErrUnmatchedTopic = fmt.Errorf("unmatched topic")
)

// JoinHandler invoked when the client joins a channel (event:_join, `js: channel.join()`).
//
// See Channel.Join
type JoinHandler func(payload any, socket *Socket) (reply any, err error)

// InHandler invoked when the client push an event to a channel (`js: channel.push(event, payload)`).
//
// See Channel.HandleIn
type InHandler func(event string, payload any, socket *Socket) (reply any, err error)

// OutHandler invoked when a broadcast message is intercepted.
//
// See Channel.HandleOut
type OutHandler func(event string, payload any, socket *Socket)

// LeaveHandler invoked when the socket leave a channel.
//
// See LeaveReason, Channel.Leave
type LeaveHandler func(socket *Socket, reason LeaveReason)

// NewChannel Defines a channel matching the given topic.
func NewChannel(topicPattern string, factory func(channel *Channel)) *Channel {
	channel := &Channel{TopicPattern: topicPattern}
	factory(channel)
	return channel
}

// Channel provide a means for bidirectional communication from clients that integrate with the pubsub layer for
// soft-realtime functionality.
type Channel struct {
	TopicPattern  string // The string pattern, for example `"room:*"`, `"users:*"`, or `"system"`
	joinHandlers  *pkg.WildcardStore[JoinHandler]
	inHandlers    *pkg.WildcardStore[InHandler]
	outHandlers   *pkg.WildcardStore[OutHandler]
	leaveHandlers *pkg.WildcardStore[LeaveHandler]
	serializer    chain.Serializer
	sockets       map[string]map[*Socket]bool
	socketsMutex  sync.RWMutex
}

// Join Handle channel joins by `topic`.
//
// To authorize a socket, return `nil, nil` or `SOME_REPLY_PAYLOAD, nil`.
//
// To refuse authorization, return `nil, reason`.
//
// Example
//
//		channel.Join("room:lobby", func Join(payload any, socket *Socket) (reply any, err error)
//	    	if !authorized(payload) {
//				err = errors.New("unauthorized")
//	       	}
//			return
//	     })
func (c *Channel) Join(topic string, handler JoinHandler) {
	if c.joinHandlers == nil {
		c.joinHandlers = &pkg.WildcardStore[JoinHandler]{}
	}
	if err := c.joinHandlers.Insert(topic, handler); err != nil {
		panic(fmt.Sprintf("[chain.socket] invalid join handler for topic. Topic: %s, Error: %s", topic, err.Error()))
	}
	return
}

// HandleIn Handle incoming `event`s.
//
// ## Example
//
//		channel.HandleIn("current_rank", func(event string, payload any, socket *Socket) (reply any, err error) {
//			// client asks for their current rank, push sent directly as a new event.
//			socket.Push("current_rank", map[string]any{"val": game.GetRank(socket.Get("user"))})
//			return
//	    })
func (c *Channel) HandleIn(event string, handler InHandler) {
	if c.inHandlers == nil {
		c.inHandlers = &pkg.WildcardStore[InHandler]{}
	}
	if err := c.inHandlers.Insert(event, handler); err != nil {
		panic(fmt.Sprintf("[chain.socket] invalid InHandler for event. Event: %s, Error: %s", event, err.Error()))
	}
}

// HandleOut Intercepts outgoing `event`s.
//
// By default, broadcasted events are pushed directly to the client, but intercepting events gives your channel a
// chance to customize the event for the client to append extra information or filter the message from being
// delivered.
//
// *Note*: intercepting events can introduce significantly more overhead if a large number of subscribers must
// customize a message since the broadcast will be encoded N times instead of a single shared encoding across all
// subscribers.
//
// ## Example
//
//		channel.HandleOut("new_msg", func(event string, payload any, socket *Socket) {
//			if obj, valid := payload.(map[string]any); valid {
//				obj["is_editable"] = User.CanEditMessage(socket.Get("user"), obj)
//	       		socket.Push("new_msg", obj)
//			}
//		})
func (c *Channel) HandleOut(event string, handler OutHandler) {
	if c.outHandlers == nil {
		c.outHandlers = &pkg.WildcardStore[OutHandler]{}
	}
	if err := c.outHandlers.Insert(event, handler); err != nil {
		panic(fmt.Sprintf("[chain] invalid OutHandler for event. Event: %s, Error: %s", event, err.Error()))
	}
}

// Leave Invoked when the socket is about to leave a Channel. See LeaveHandler
func (c *Channel) Leave(topic string, handler LeaveHandler) {
	if c.leaveHandlers == nil {
		c.leaveHandlers = &pkg.WildcardStore[LeaveHandler]{}
	}
	if err := c.leaveHandlers.Insert(topic, handler); err != nil {
		panic(fmt.Sprintf("[chain] invalid LeaveHandler for topic. Topic: %s, Error: %s", topic, err.Error()))
	}
}

// Broadcast on the pubsub server with the given topic, event and payload.
func (c *Channel) Broadcast(topic string, event string, payload any) (err error) {
	broadcast := newMessage(MessageTypeBroadcast, topic, event, payload)
	defer deleteMessage(broadcast)

	var bytes []byte
	if bytes, err = c.serializer.Encode(broadcast); err != nil {
		return
	}
	err = pubsub.Broadcast(topic, bytes)
	return
}

// LocalBroadcast on the pubsub server with the given topic, event and payload.
func (c *Channel) LocalBroadcast(topic string, event string, payload any) (err error) {
	broadcast := newMessage(MessageTypeBroadcast, topic, event, payload)
	pubsub.LocalBroadcast(topic, broadcast)
	return
}

// Dispatch Hook invoked by pubsub dispatch.
func (c *Channel) Dispatch(topic string, msg any, from string) {
	var message *Message
	var valid bool
	var payload []byte
	isByteArray := false

	if payload, valid = msg.([]byte); valid {
		isByteArray = true
		message = newMessageAny()
		if _, err := c.serializer.Decode(payload, message); err != nil {
			slog.Debug(
				"[chain.socket] could not decode serialized data",
				slog.Any("Error", err),
				slog.Any("Payload", payload),
				slog.String("Topic", topic),
			)

			deleteMessage(message)
			return
		}
	} else if message, valid = msg.(*Message); !valid {
		return
	}

	// get sockets
	c.socketsMutex.RLock()
	var sockets []*Socket
	if len(c.sockets) > 0 {
		if ss, exist := c.sockets[topic]; exist {
			for socket, _ := range ss {
				sockets = append(sockets, socket)
			}
		}
	}
	c.socketsMutex.RUnlock()

	if len(sockets) == 0 {
		return
	}

	// check for out handler (intercept)
	if c.outHandlers != nil {
		if handler := c.outHandlers.Match(message.Event); handler != nil {
			for _, socket := range sockets {
				handler(message.Event, message.Payload, socket)
			}
			// intercepted
			return
		}
	}

	// fastlane (not intercepted, single encode for all sockets)

	if !isByteArray {
		var err error
		if payload, err = c.serializer.Encode(message); err != nil {
			return
		}
	}

	for _, socket := range sockets {
		socket.Send(payload)
	}
}

// validate @todo checks if all handlers are configured correctly
func (c *Channel) validate() (err error) {
	return nil
}

func (c *Channel) handleJoin(topic string, payload any, socket *Socket) (reply any, err error) {
	defer func() {
		if rcv := recover(); rcv != nil {
			//pc, file, line, _ := runtime.Caller(2)
			//lib.Warning("initialization process failed %s[%q] %#v at %s[%s:%d]", process.self, name, rcv, runtime.FuncForPC(pc).Name(), fn, line)
			err = ErrJoinCrashed
		}
	}()

	if c.joinHandlers != nil {
		if handler := c.joinHandlers.Match(topic); handler != nil {
			if reply, err = handler(payload, socket); err == nil {
				// subscribe topic and configure fastlane
				pubsub.Subscribe(topic, c)

				c.socketsMutex.Lock()
				defer c.socketsMutex.Unlock()

				if c.sockets == nil {
					c.sockets = map[string]map[*Socket]bool{}
				}
				if _, exist := c.sockets[socket.Topic()]; !exist {
					c.sockets[socket.Topic()] = map[*Socket]bool{}
				}
				c.sockets[socket.Topic()][socket] = true
				return
			}
		}
	}

	err = ErrUnmatchedTopic
	return
}

func (c *Channel) handleLeave(socket *Socket, reason LeaveReason) {
	if socket.channel == c {
		socket.channel = nil

		topic := socket.Topic()

		pubsub.Unsubscribe(topic, c)

		// remove socket reference on channel
		c.socketsMutex.Lock()
		defer c.socketsMutex.Unlock()
		if len(c.sockets) == 0 {
			return
		}

		delete(c.sockets[topic], socket)
		if c.leaveHandlers != nil {
			if handler := c.leaveHandlers.Match(topic); handler != nil {
				handler(socket, reason)
			}
		}
	}
	return
}

func (c *Channel) handleIn(event string, payload any, socket *Socket) (reply any, err error) {
	if c.inHandlers == nil {
		err = ErrUnmatchedTopic
		return
	}

	handler := c.inHandlers.Match(event)
	if handler == nil {
		err = ErrUnmatchedTopic
	} else {
		reply, err = handler(event, payload, socket)
	}

	return
}
