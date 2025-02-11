package socket

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/pkg"
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
	channels      *pkg.WildcardStore[*Channel]
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex
}

func (h *Handler) Configure(router *chain.Router, endpoint string) {

	ClientJsHandler(router, endpoint)

	if h.Options == nil {
		h.Options = map[string]any{}
	}

	if h.OnConfig != nil {
		if err := h.OnConfig(h, router, endpoint); err != nil {
			panic(fmt.Sprintf("[chain.socket] socket handler config error. Error: %s", err.Error()))
		}
	}

	h.sessions = map[string]*Session{}

	if len(h.Channels) == 0 {
		panic(fmt.Sprintf("[chain.socket] is necessary to inform the channels of this socket. Endpoint: %s", endpoint))
	}

	if h.Serializer == nil {
		h.Serializer = defaultSerializer
	}

	h.channels = &pkg.WildcardStore[*Channel]{}

	for _, channel := range h.Channels {
		if err := h.channels.Insert(channel.topicPattern, channel); err != nil {
			panic(fmt.Sprintf("[chain.socket] invalid channel for topic. TopicPattern: %s, Error: %s", channel.topicPattern, err.Error()))
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
		id:       socketId,
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

		message := getMessageAny()
		if _, err := h.Serializer.Decode(payload, message); err != nil {
			slog.Debug(
				"[chain.socket] could not decode serialized data",
				slog.Any("Error", err),
				slog.Any("Payload", payload),
			)

			putMessage(message)
			return
		}

		switch message.Event {
		case "_join":
			h.handleJoin(message, session)
		case "_leave":
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
		slog.Info(
			"[chain.socket] ignoring unmatched topic",
			slog.Any("session_id", session.Id()),
			slog.String("Topic", topic),
		)

		h.pushIgnore(message, session, ErrUnmatchedTopic)
		return
	}
	socket := session.GetSocket(topic)
	if socket != nil {
		slog.Info(
			"[chain.socket] duplicated channel join. closing existing channel for new join",
			slog.Any("session_id", session.Id()),
			slog.String("Topic", topic),
		)

		// remove from transport
		session.deleteSocket(topic)

		if socket.status != StatusLeaving {
			if socket.channel != nil {
				socket.channel.handleLeave(socket, LeaveReasonRejoin)
			}

			if socket.joinRef != message.JoinRef {
				reply := getMessage(MessageTypePush, topic, "_close", nil)
				reply.Ref = socket.ref
				reply.JoinRef = socket.joinRef
				h.push(reply, session)
			}

			putSocket(socket)
		}
	}

	socket = getSocket(message.Ref, message.JoinRef, topic, channel, session, h)

	socket.Params = session.Params

	payload, err := channel.handleJoin(topic, message.Payload, socket)
	if err != nil {
		putSocket(socket)
		h.pushIgnore(message, session, err)
		return
	}

	socket.status = StatusJoined

	session.setSocket(topic, socket)

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

		putSocket(socket)
	}

	message.Kind = MessageTypeReply
	message.Status = ReplyStatusCodeOk

	h.push(message, info)
}

func (h *Handler) handleMessage(message *Message, session *Session) {
	topic := message.Topic
	socket := session.GetSocket(topic)
	if socket == nil {
		slog.Info(
			"[chain.socket] ignoring unmatched topic",
			slog.Any("session_id", session.Id()),
			slog.String("Topic", topic),
		)

		h.pushIgnore(message, session, ErrUnmatchedTopic)
		return
	}

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
	} else {
		putMessage(message)
	}
}

func (h *Handler) handleClose(info *Session) {
	h.sessionsMutex.Lock()
	delete(h.sessions, info.Id())
	h.sessionsMutex.Unlock()

	info.socketsMutex.Lock()
	defer info.socketsMutex.Unlock()

	if info.socketsByTopic != nil {
		for _, socket := range info.socketsByTopic {
			if socket.status != StatusLeaving {
				if socket.channel != nil {
					socket.channel.handleLeave(socket, LeaveReasonClose)
				}

				putSocket(socket)
			}
		}
	}
}

func (h *Handler) handleHeartbeat(message *Message, info *Session) {

}

func (h *Handler) push(message *Message, info *Session) {
	defer putMessage(message)
	var bytes []byte
	var err error
	if bytes, err = h.Serializer.Encode(message); err != nil {
		slog.Debug(
			"[chain.socket] could not encode message",
			slog.Any("Error", err),
			slog.Int("Kind", int(message.Kind)),
			slog.Int("JoinRef", message.JoinRef),
			slog.Int("Ref", message.Ref),
			slog.Int("Status", message.Status),
			slog.String("Topic", message.Topic),
			slog.String("Event", message.Event),
			slog.Any("Payload", message.Payload),
		)
		return
	}
	info.Push(bytes)
}

func (h *Handler) pushIgnore(message *Message, info *Session, reason error) {
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

func getSocket(ref int, joinRef int, topic string, channel *Channel, info *Session, handler *Handler) *Socket {
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

func putSocket(socket *Socket) {
	socket.topic = ""
	socket.channel = nil
	socket.session = nil
	socket.handler = nil
	socket.data = nil
	socket.status = StatusRemoved
	socketPool.Put(socket)
}
