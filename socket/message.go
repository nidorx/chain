package socket

import "sync"

type MessageType int

const (
	ReplyStatusCodeOk    = 0
	ReplyStatusCodeError = 1
	MessageTypePush      = MessageType(0)
	MessageTypeReply     = MessageType(1) // Defines a reply Message sent from channels to Transport.
	MessageTypeBroadcast = MessageType(2) // Defines a Message sent from pubsub to channels and vice-versa.
)

// Message Defines a message dispatched over transport to channels and vice-versa.
type Message struct {
	Kind    MessageType `json:"k,omitempty"` // Type of message
	JoinRef int         `json:"j,omitempty"` // The unique number ref when joining
	Ref     int         `json:"r,omitempty"` // The unique number ref
	Status  int         `json:"s,omitempty"` // The reply status
	Topic   string      `json:"t,omitempty"` // The string topic or topic:subtopic pair namespace, for example "messages", "messages:123"
	Event   string      `json:"e,omitempty"` // The string event name, for example "_join"
	Payload any         `json:"p,omitempty"` // The Message payload
}

var messagePool = &sync.Pool{
	New: func() any {
		return &Message{}
	},
}

func getMessageAny() *Message {
	return messagePool.Get().(*Message)
}

func getMessage(kind MessageType, topic string, event string, payload any) *Message {
	m := messagePool.Get().(*Message)
	m.Kind = kind
	m.Topic = topic
	m.Event = event
	m.Payload = payload
	return m
}

func putMessage(m *Message) {
	m.Payload = nil
	m.Event = ""
	m.Topic = ""
	m.Ref = 0
	m.JoinRef = 0
	m.Status = 0
	messagePool.Put(m)
}
