package socket

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/pkg"
	"github.com/nidorx/chain/pubsub"
)

// LeaveReason reasons why a LeaveHandler is invoked
type LeaveReason int

const (
	LeaveReasonLeave  LeaveReason = 0 // Client called _leave event (channel.leave()).
	LeaveReasonRejoin LeaveReason = 1 // Client called _join and there is already an active socket for the same topic
	LeaveReasonClose  LeaveReason = 2 // Connection lost and session is terminated. See Session.ScheduleShutdown
)

var (
	ErrJoinCrashed       = errors.New("join crashed")
	ErrUnmatchedTopic    = errors.New("unmatched topic")
	ErrJoinWildcardTopic = errors.New("joining topics with wildcard is not allowed")
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
	channel := &Channel{topicPattern: topicPattern}
	factory(channel)
	return channel
}

// Channel provide a means for bidirectional communication from clients that integrate with the pubsub layer for
// soft-realtime functionality.
type Channel struct {
	topicPattern   string // The string pattern, for example `"room:*"`, `"users:*"`, or `"system"`
	joinHandlers   *pkg.WildcardStore[JoinHandler]
	inHandlers     *pkg.WildcardStore[InHandler]
	outHandlers    *pkg.WildcardStore[OutHandler]
	leaveHandlers  *pkg.WildcardStore[LeaveHandler]
	serializer     chain.Serializer
	socketsMutex   sync.RWMutex
	socketsByTopic map[string]map[string]*Socket
}

func (c *Channel) TopicPattern() string {
	return c.topicPattern
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

// Broadcast to all sockets on the pubsub cluster with the given topic, event and payload.
func (c *Channel) Broadcast(topic string, event string, payload any) error {
	message := getMessage(MessageTypeBroadcast, topic, event, payload)
	defer putMessage(message)

	if bytes, err := c.serializer.Encode(message); err != nil {
		return err
	} else {
		return pubsub.Broadcast("ch:"+topic, bytes)
	}
}

// LocalBroadcast to all sockets on local server with the given topic, event and payload.
func (c *Channel) LocalBroadcast(topic string, event string, payload any) error {
	message := getMessage(MessageTypeBroadcast, topic, event, payload)
	defer putMessage(message)

	if bytes, err := c.serializer.Encode(message); err != nil {
		return err
	} else {
		pubsub.LocalBroadcast("ch:"+topic, bytes)
		return nil
	}
}

// Subscribe to the pubsub topic automaticaly pushing messages to the joined clients
func (c *Channel) Subscribe(topicPattern, event string) {
	pubsub.Subscribe(topicPattern, pubsub.DispatcherFunc(func(topic string, pubsubPayload []byte, from string) {
		var payload any
		if err := json.Unmarshal(pubsubPayload, &payload); err != nil {
			slog.Warn(
				"[chain.socket] failed to decode pubsub message",
				slog.Any("error", err),
				slog.String("from", from),
				slog.String("event", event),
				slog.String("pubsubTopic", topicPattern),
				slog.String("topic", topic),
				slog.String("message", string(pubsubPayload)),
			)
			return
		}

		c.dispatch(topic, getMessage(MessageTypeBroadcast, topic, event, payload), from)
	}))
}

// Dispatch Hook invoked by pubsub dispatch.
func (c *Channel) Dispatch(topic string, channelMessageEncoded []byte, from string) {
	var message = getMessageAny()

	topic = strings.TrimPrefix(topic, "ch:")

	if _, err := c.serializer.Decode(channelMessageEncoded, message); err != nil {
		slog.Debug(
			"[chain.socket] could not decode serialized data",
			slog.Any("Error", err),
			slog.String("topic", topic),
			slog.String("from", from),
		)
		putMessage(message)
		return
	}

	c.dispatch(topic, message, from)
}

func (c *Channel) dispatch(topic string, message *Message, _ string) {

	defer putMessage(message)

	// get sockets
	c.socketsMutex.RLock()
	var sockets []*Socket
	if len(c.socketsByTopic) > 0 {
		if ss, exist := c.socketsByTopic[topic]; exist {
			for _, socket := range ss {
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

	encoded, err := c.serializer.Encode(message)
	if err != nil {
		return
	}

	for _, socket := range sockets {
		socket.Send(encoded)
	}
}

func (c *Channel) handleJoin(topic string, params any, socket *Socket) (reply any, err error) {
	defer func() {
		if rcv := recover(); rcv != nil {
			//pc, file, line, _ := runtime.Caller(2)
			//lib.Warning("initialization process failed %s[%q] %#v at %s[%s:%d]", process.self, name, rcv, runtime.FuncForPC(pc).Name(), fn, line)
			err = ErrJoinCrashed
		}
	}()

	if c.joinHandlers != nil {
		if handler := c.joinHandlers.Match(topic); handler != nil {
			if reply, err = handler(params, socket); err == nil {

				// subscribe channel topic and configure fastlane
				// prefix "ch:" to be socket exclusive events
				pubsub.Subscribe("ch:"+topic, c)

				c.socketsMutex.Lock()
				defer c.socketsMutex.Unlock()

				if c.socketsByTopic == nil {
					c.socketsByTopic = map[string]map[string]*Socket{}
				}
				if _, exist := c.socketsByTopic[socket.Topic()]; !exist {
					c.socketsByTopic[socket.Topic()] = map[string]*Socket{}
				}
				c.socketsByTopic[socket.Topic()][socket.Id()] = socket
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

		pubsub.Unsubscribe("ch:"+topic, c)

		// remove socket reference on channel
		c.socketsMutex.Lock()
		defer c.socketsMutex.Unlock()
		if len(c.socketsByTopic) == 0 {
			return
		}

		delete(c.socketsByTopic[topic], socket.Id())
		if c.leaveHandlers != nil {
			if handler := c.leaveHandlers.Match(topic); handler != nil {
				handler(socket, reason)
			}
		}
	}
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
