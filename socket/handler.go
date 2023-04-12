package socket

import (
	"github.com/rs/zerolog/log"
	"github.com/syntax-framework/chain"
	"github.com/syntax-framework/chain/pkg"
	"sync"
)

func _l(msg string) string {
	return "[chain.socket] " + msg
}

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
	channels      *pkg.WildcardStore[*Channel]
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex
}

func (h *Handler) Configure(router *chain.Router, endpoint string) {

	ClientJsAddHandler(router)

	if h.Options == nil {
		h.Options = map[string]any{}
	}

	if h.OnConfig != nil {
		if err := h.OnConfig(h, router, endpoint); err != nil {
			log.Panic().Stack().Err(err).Caller(1).
				Msg(_l("socket handler config error"))
		}
	}

	h.sessions = map[string]*Session{}

	if len(h.Channels) == 0 {
		log.Panic().Caller(1).
			Msg(_l("is necessary to inform the channels of this socket"))
	}

	if h.Serializer == nil {
		h.Serializer = defaultSerializer
	}

	h.channels = &pkg.WildcardStore[*Channel]{}

	for _, channel := range h.Channels {
		if err := h.channels.Insert(channel.TopicPattern, channel); err != nil {
			log.Panic().Stack().Err(err).Caller(1).
				Str("TopicPattern", channel.TopicPattern).
				Msg(_l("invalid channel for topic"))
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
			log.Debug().Err(err).Caller(1).
				Bytes("payload", payload).
				Msg(_l("could not decode serialized data"))

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
func (h *Handler) handleJoin(message *Message, session *Session) {
	topic := message.Topic
	channel := h.getChannel(topic)
	if channel == nil {
		log.Info().Caller(1).
			Str("topic", topic).
			Str("socket_id", session.SocketId()).
			Msg(_l("ignoring unmatched topic"))

		h.pushIgnore(message, session, ErrUnmatchedTopic)
		return
	}
	socket := session.GetSocket(topic)
	if socket != nil {
		log.Info().
			Str("topic", topic).
			Str("socket_id", session.SocketId()).
			Msg(_l("duplicate channel join. closing existing channel for new join"))

		// remove from transport
		session.deleteSocket(topic)

		if socket.status != StatusLeaving {
			if socket.channel != nil {
				socket.channel.handleLeave(socket, LeaveReasonRejoin)
			}

			if socket.joinRef != message.JoinRef {
				reply := newMessage(MessageTypePush, topic, "stx_close", nil)
				reply.Ref = socket.ref
				reply.JoinRef = socket.joinRef
				h.push(reply, session)
				deleteMessage(reply)
			}

			deleteSocket(socket)
		}
	}

	socket = newSocket(message.Ref, message.JoinRef, topic, channel, session, h)

	socket.Params = session.Params

	payload, err := channel.handleJoin(topic, message.Payload, socket)
	if err != nil {
		deleteSocket(socket)
		h.pushIgnore(message, session, err)
		return
	}

	socket.status = StatusJoined

	session.setSocket(topic, socket)

	defer deleteMessage(message)
	message.Kind = MessageTypeReply
	message.Status = ReplyStatusCodeOk
	message.Payload = payload

	h.push(message, session)
}

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

func (h *Handler) handleMessage(message *Message, session *Session) {
	topic := message.Topic
	socket := session.GetSocket(topic)
	if socket == nil {
		log.Info().
			Str("topics", topic).
			Str("socket_id", session.SocketId()).
			Msg(_l("ignoring unmatched topic"))

		h.pushIgnore(message, session, ErrUnmatchedTopic)
		return
	}

	defer deleteMessage(message)

	channel := socket.channel
	payload, err := channel.handleIn(message.Event, message.Payload, socket)
	if err != nil {
		message.Kind = MessageTypeReply
		message.Status = ReplyStatusCodeError
		message.Payload = payload
		h.push(message, session)
	} else if payload != nil {
		message.Kind = MessageTypeReply
		message.Status = ReplyStatusCodeOk
		message.Payload = payload
		h.push(message, session)
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
		log.Debug().Err(err).Caller(1).
			Int("msg.Kind", int(reply.Kind)).
			Int("msg.JoinRef", reply.JoinRef).
			Int("msg.Ref", reply.Ref).
			Int("msg.Status", reply.Status).
			Str("msg.Topic", reply.Topic).
			Str("msg.Event", reply.Event).
			Interface("msg.Payload", reply.Payload).
			Msg(_l("could not encode message"))
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
