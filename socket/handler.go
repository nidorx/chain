package socket

import (
	"fmt"
	"github.com/syntax-framework/chain"
	"github.com/syntax-framework/chain/lib"
	"sync"
)

var (
	defaultSerializer = &MessageSerializer{}
	socketPool        = &sync.Pool{
		New: func() any {
			return &Socket{}
		},
	}
)

type ConnectHandler func(session *Session) error

type ConfigHandler func(handler *Handler, router *chain.Router, endpoint string) error

// Handler A socket implementation that multiplexes messages over channels.
//
// Handler is used as a module for establishing and maintaining the socket state via the Session and Socket struct.
//
// Once connected to a socket, incoming and outgoing events are routed to Channel. The incoming client data is routed
// to channels via transports. It is the responsibility of the Handler to tie Transport and Channel together.
type Handler struct {
	Options       map[string]any   // Permite receber opções que estrão acessíveis
	Channels      []*Channel       // Channels in this socket
	Transports    []Transport      // Configured Transports
	Serializer    chain.Serializer // Serializer definido para o Transport
	OnConfig      ConfigHandler    // Called by Handler.Configure
	OnConnect     ConnectHandler   // Called when client try to connect on a Transport
	channels      *lib.WildcardStore[*Channel]
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex
}

func (h *Handler) Configure(router *chain.Router, endpoint string) {

	if h.Options == nil {
		h.Options = map[string]any{}
	}

	if h.OnConfig != nil {
		if err := h.OnConfig(h, router, endpoint); err != nil {
			panic(any(fmt.Sprintf("socket handler config error. Cause: %s", err.Error())))
		}
	}

	h.sessions = map[string]*Session{}

	if len(h.Channels) == 0 {
		panic(any("It is necessary to inform the channels of this socket"))
	}

	if h.Serializer == nil {
		h.Serializer = defaultSerializer
	}

	h.channels = &lib.WildcardStore[*Channel]{}

	for _, channel := range h.Channels {
		if err := h.channels.Insert(channel.TopicPattern, channel); err != nil {
			panic(any(fmt.Sprintf("invalid channel with topic %s. Cause: %s", channel.TopicPattern, err.Error())))
		}
		channel.serializer = h.Serializer
	}

	if len(h.Transports) == 0 {
		h.Transports = []Transport{&TransportSSE{}}
	}

	for _, transport := range h.Transports {
		transport.Configure(h, router, endpoint)
	}
}

// Connect invoked by Transport, initializes a new session
func (h *Handler) Connect(endpoint string, params map[string]string) (session *Session, err error) {
	socketId := chain.NewUID()
	messages := make(chan []byte, 32)

	session = &Session{
		Params:   params,
		Options:  h.Options,
		socketId: socketId,
		endpoint: endpoint,
		handler:  h,
		closed:   false,
		messages: messages,
	}

	if h.OnConnect != nil {
		err = h.OnConnect(session)
	}

	if err == nil {
		h.sessionsMutex.Lock()
		h.sessions[socketId] = session
		h.sessionsMutex.Unlock()
	}

	return
}

// Resume used by Transport, tries to recover the session if it still alive
func (h *Handler) Resume(socketId string) *Session {
	h.sessionsMutex.RLock()
	session, exist := h.sessions[socketId]
	h.sessionsMutex.RUnlock()

	if exist {
		session.StopScheduledShutdown()
		if !session.closed {
			return session
		}
	}

	return nil
}

// Dispatch Processes messages from Transport (client)
func (h *Handler) Dispatch(payload []byte, session *Session) {
	go func() {
		// @todo: goroutine using ants
		// @todo: defer recovery

		message := newMessageAny()
		if _, err := h.Serializer.Decode(payload, message); err != nil {
			// @todo: Log
			deleteMessage(message)
			return
		}

		switch message.Event {
		case "stx_join":
			h.handleJoin(message, session)
		case "stx_leave":
			h.handleLeave(message, session)
		case "heartbeat":
			h.handleHeartbeat(message, session)
		default:
			h.handleMessage(message, session)
		}
	}()
}

// handleJoin Joins the channel in socket with authentication payload.
func (h *Handler) handleJoin(message *Message, info *Session) {
	topic := message.Topic
	channel := h.getChannel(topic)
	if channel == nil {
		println(fmt.Sprintf("Ignoring unmatched topic '%s'. SocketId: %s", topic, info.socketId))
		h.pushIgnore(message, info, ErrUnmatchedTopic)
		return
	}
	socket := info.GetSocket(topic)
	if socket != nil {
		println(fmt.Sprintf("Duplicate channel join for topic '%s'. SocketId: %s. Closing existing channel for new join.", topic, info.socketId))

		// remove from transport
		info.deleteSocket(topic)

		if socket.status != StatusLeaving {
			if socket.channel != nil {
				socket.channel.handleLeave(socket, LeaveReasonRejoin)
			}

			if socket.joinRef != message.JoinRef {
				reply := newMessage(MessageTypePush, topic, "stx_close", nil)
				reply.Ref = socket.ref
				reply.JoinRef = socket.joinRef
				h.push(reply, info)
				deleteMessage(reply)
			}

			deleteSocket(socket)
		}
	}

	socket = newSocket(message.Ref, message.JoinRef, topic, channel, info, h)

	socket.Params = info.Params

	payload, err := channel.handleJoin(topic, message.Payload, socket)
	if err != nil {
		deleteSocket(socket)
		h.pushIgnore(message, info, err)
		return
	}

	socket.status = StatusJoined

	// @todo: pubsub.subscribe(pubsub_server, topic, metadata: fastlane)

	info.setSocket(topic, socket)

	defer deleteMessage(message)
	message.Kind = MessageTypeReply
	message.Status = ReplyStatusCodeOk
	message.Payload = payload

	h.push(message, info)
}

// handleLeave faz processamento da solicitação de saída do channel
func (h *Handler) handleLeave(message *Message, info *Session) {
	topic := message.Topic
	socket := info.GetSocket(topic)
	if socket != nil {
		socket.status = StatusLeaving

		// remove from transport
		info.deleteSocket(topic)

		if socket.channel != nil {
			socket.channel.handleLeave(socket, LeaveReasonLeave)
		}

		deleteSocket(socket)
	}

	defer deleteMessage(message)
	message.Kind = MessageTypeReply
	message.Status = ReplyStatusCodeOk

	h.push(message, info)
}

func (h *Handler) handleMessage(message *Message, info *Session) {
	topic := message.Topic
	socket := info.GetSocket(topic)
	if socket == nil {
		println(fmt.Sprintf("Ignoring unmatched topic '%s' socket. SocketId: %s", topic, info.socketId))
		h.pushIgnore(message, info, ErrUnmatchedTopic)
		return
	}

	defer deleteMessage(message)

	channel := socket.channel
	payload, err := channel.handleIn(message.Event, message.Payload, socket)
	if err != nil {
		message.Kind = MessageTypeReply
		message.Status = ReplyStatusCodeError
		message.Payload = payload
		h.push(message, info)
	} else if payload != nil {
		message.Kind = MessageTypeReply
		message.Status = ReplyStatusCodeOk
		message.Payload = payload
		h.push(message, info)
	}
}

func (h *Handler) handleClose(info *Session) {
	h.sessionsMutex.Lock()
	delete(h.sessions, info.SocketId())
	h.sessionsMutex.Unlock()

	info.socketsMutex.Lock()
	defer info.socketsMutex.Unlock()

	if info.sockets != nil {
		for _, socket := range info.sockets {
			if socket.status != StatusLeaving {
				if socket.channel != nil {
					socket.channel.handleLeave(socket, LeaveReasonClose)
				}

				deleteSocket(socket)
			}
		}
	}
}

func (h *Handler) handleHeartbeat(message *Message, info *Session) {

}

func (h *Handler) push(reply *Message, info *Session) {
	var bytes []byte
	var err error
	if bytes, err = h.Serializer.Encode(reply); err != nil {
		// @todo: log
		return
	}
	info.Push(bytes)
}

func (h *Handler) pushIgnore(message *Message, info *Session, reason error) {
	defer deleteMessage(message)
	message.Kind = MessageTypeReply
	message.Status = ReplyStatusCodeError
	message.Payload = map[string]string{"reason": reason.Error()}
	h.push(message, info)
}

func (h *Handler) getChannel(topic string) *Channel {
	if item := h.channels.Match(topic); item != nil {
		return item
	}
	return nil
}

func newSocket(ref int, joinRef int, topic string, channel *Channel, info *Session, handler *Handler) *Socket {
	socket := socketPool.Get().(*Socket)
	socket.ref = ref
	socket.joinRef = joinRef
	socket.topic = topic
	socket.channel = channel
	socket.session = info
	socket.handler = handler
	socket.status = StatusJoining
	socket.data = map[string]any{}
	return socket
}

func deleteSocket(socket *Socket) {
	socket.topic = ""
	socket.channel = nil
	socket.session = nil
	socket.handler = nil
	socket.data = nil
	socket.status = StatusRemoved
	socketPool.Put(socket)
}
