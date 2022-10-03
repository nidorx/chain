package socket

import (
	"sync"
	"time"
)

// Session used by Transport, communication interface between Transport and Channel.
//
// Keeps an active session on the server. Transport should invoke ScheduleShutdown method when user connection drops
type Session struct {
	Params        map[string]string  // Initialization parameters, received at connection time
	Options       map[string]any     // Reference to Handler.Options
	closed        bool               // Session still active?
	handler       *Handler           // Reference to the Handler of this session
	socketId      string             // Session id
	endpoint      string             // Path to socket endpoint
	sockets       map[string]*Socket // Socket by topic
	messages      chan []byte        // Messages that will be delivered to the client
	shutdown      *time.Timer        // Session termination timeout
	socketsMutex  sync.RWMutex
	shutdownMutex sync.Mutex
}

// SocketId Session id
func (s *Session) SocketId() string {
	return s.socketId
}

// Endpoint Path to socket endpoint
func (s *Session) Endpoint() string {
	return s.endpoint
}

// GetSocket get the Socket associated with the given topic
func (s *Session) GetSocket(topic string) *Socket {
	s.socketsMutex.RLock()
	defer s.socketsMutex.RUnlock()

	if s.sockets != nil {
		if socket, exist := s.sockets[topic]; exist {
			return socket
		}
	}

	return nil
}

// Push message to client
func (s *Session) Push(bytes []byte) {
	select {
	case s.messages <- bytes:
	default:

	}
}

// Dispatch message to Channel
func (s *Session) Dispatch(message []byte) {
	s.StopScheduledShutdown()
	if !s.closed {
		s.handler.Dispatch(message, s)
	}
}

// StopScheduledShutdown cancels the final termination of that session.
//
// Invoked by the Handler.Resume method
func (s *Session) StopScheduledShutdown() {
	if s.shutdown != nil {
		s.shutdownMutex.Lock()
		defer s.shutdownMutex.Unlock()
		if s.shutdown != nil {
			s.shutdown.Stop()
			s.shutdown = nil
		}
	}
}

// ScheduleShutdown schedules the termination of this session on the server.
// In case of a user reconnection, invoke the StopScheduledShutdown or Handler.Resume methods
func (s *Session) ScheduleShutdown(after time.Duration) {
	s.shutdownMutex.Lock()
	defer s.shutdownMutex.Unlock()
	if s.shutdown == nil {
		s.shutdown = time.AfterFunc(after, func() {
			s.shutdownMutex.Lock()
			defer s.shutdownMutex.Unlock()
			if s.shutdown == nil {
				return
			}
			s.close()
		})
	}
}

func (s *Session) setSocket(topic string, socket *Socket) {
	s.socketsMutex.Lock()
	defer s.socketsMutex.Unlock()

	if s.sockets == nil {
		s.sockets = map[string]*Socket{}
	}
	s.sockets[topic] = socket
}

func (s *Session) deleteSocket(topic string) {
	s.socketsMutex.Lock()
	defer s.socketsMutex.Unlock()
	if s.sockets != nil {
		delete(s.sockets, topic)
	}
}

// close invoked by ScheduleShutdown when session is permanently terminated
func (s *Session) close() {
	s.closed = true
	s.shutdown = nil
	s.handler.handleClose(s)
	s.sockets = nil
}
