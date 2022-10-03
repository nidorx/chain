package socket

import "fmt"

type Status int

const (
	StatusJoining = Status(0)
	StatusJoined  = Status(1)
	StatusLeaving = Status(2)
	StatusRemoved = Status(3)
)

var (
	ErrSocketNotJoined = fmt.Errorf("socket not joined")
)

// Socket Channel integration.
//
// Allows the channel to manage socket state data through the Socket.Set and Socket.Get
type Socket struct {
	Params  map[string]string // Initialization parameters, received at connection time.
	ref     int
	joinRef int
	topic   string
	channel *Channel
	session *Session
	data    map[string]any
	status  Status
	handler *Handler
}

func (s *Socket) Id() string {
	return s.session.socketId
}

func (s *Socket) Endpoint() string {
	return s.session.endpoint
}

func (s *Socket) Topic() string {
	return s.topic
}

func (s *Socket) Status() Status {
	return s.status
}

func (s *Socket) Session() *Session {
	return s.session
}

// Get a value from Socket (server side only)
func (s *Socket) Get(key string) (value any) {
	return s.data[key]
}

// Set a value on Socket (server side only)
func (s *Socket) Set(key string, value any) {
	s.data[key] = value
}

// Push message to client
func (s *Socket) Push(event string, payload any) (err error) {
	if s.status != StatusJoined {
		// can only be called after the socket has finished joining.
		return ErrSocketNotJoined
	}

	message := newMessage(MessageTypePush, s.topic, event, payload)
	message.JoinRef = s.joinRef
	defer deleteMessage(message)

	var encoded []byte
	if encoded, err = s.handler.Serializer.Encode(message); err != nil {
		return
	}
	s.session.Push(encoded)
	return
}

// Send encoded message to client
func (s *Socket) Send(bytes []byte) error {
	if s.status != StatusJoined {
		// can only be called after the socket has finished joining.
		return ErrSocketNotJoined
	}
	s.session.Push(bytes)
	return nil
}

// Broadcast an event to all subscribers of the socket topic.
func (s *Socket) Broadcast(event string, payload any) (err error) {
	if s.status != StatusJoined {
		// can only be called after the socket has finished joining.
		return ErrSocketNotJoined
	}
	return s.channel.Broadcast(s.Topic(), event, payload)
}
