package socket

import (
	"encoding/json"
	"errors"
)

type MessageSerializer struct{}

func (s *MessageSerializer) Encode(v any) (data []byte, err error) {
	var msg *Message
	var valid bool
	if msg, valid = v.(*Message); !valid {
		err = errors.New("can only serialize *Message")
		return
	}

	// Push 		= [kind, joinRef, ref,  topic, event, payload]
	// Reply 		= [kind, joinRef, ref, status,        payload]
	// Broadcast 	= [kind,                topic, event, payload]
	out := []any{msg.Kind}

	if msg.Kind != MessageTypeBroadcast {
		out = append(out, msg.JoinRef, msg.Ref)
	}

	if msg.Kind == MessageTypeReply {
		out = append(out, msg.Status)
	} else {
		out = append(out, msg.Topic)
	}

	if msg.Kind != MessageTypeReply {
		out = append(out, msg.Event)
	}

	if msg.Payload != nil {
		out = append(out, msg.Payload)
	}

	return json.Marshal(out)
}

func (s *MessageSerializer) Decode(data []byte, v any) (out any, err error) {
	var valid bool
	var msg *Message
	if msg, valid = v.(*Message); !valid {
		err = errors.New("can only deserialize *Message")
		return
	}
	out = msg

	// Push 		= [kind, joinRef, ref,  topic, event, payload]
	// Reply 		= [kind, joinRef, ref, status,        payload]
	// Broadcast 	= [kind,                topic, event, payload]
	arr := []any{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}

	if len(arr) < 4 {
		return nil, errors.New("invalid Message size")
	}

	// Kind
	if value, ok := arr[0].(float64); !ok {
		return nil, errors.New("invalid Message.Kind")
	} else {
		msg.Kind = MessageType(value)
	}

	// JoinRef | Topic
	if msg.Kind == MessageTypeBroadcast {
		// Topic
		if value, ok := arr[1].(string); !ok {
			return nil, errors.New("invalid Message.Topic")
		} else {
			msg.Topic = value
		}
	} else {
		// JoinRef
		if value, ok := arr[1].(float64); !ok {
			return nil, errors.New("invalid Message.JoinRef")
		} else {
			msg.JoinRef = int(value)
		}
	}

	// Ref | Event
	if msg.Kind == MessageTypeBroadcast {
		// Event
		if value, ok := arr[2].(string); !ok {
			return nil, errors.New("invalid Message.Event")
		} else {
			msg.Event = value
		}
	} else {
		// ref
		if value, ok := arr[2].(float64); !ok {
			return nil, errors.New("invalid Message.Ref")
		} else {
			msg.Ref = int(value)
		}
	}

	// Topic | Payload | Status
	if msg.Kind == MessageTypeReply {
		// Status
		if value, ok := arr[3].(float64); !ok {
			return nil, errors.New("invalid Message.Status")
		} else {
			msg.Status = int(value)
		}
	} else if msg.Kind == MessageTypeBroadcast {
		// Payload
		msg.Payload = arr[3]
	} else {
		// Topic
		if value, ok := arr[3].(string); !ok {
			return nil, errors.New("invalid Message.Topic")
		} else {
			msg.Topic = value
		}
	}

	// Event | Payload
	if msg.Kind == MessageTypeBroadcast {
		return // invalid message, ignore
	}

	if msg.Kind == MessageTypeReply {
		if len(arr) > 4 {
			msg.Payload = arr[4]
		}
	} else {
		if len(arr) < 5 {
			return nil, errors.New("invalid Message size")
		}
		if value, ok := arr[4].(string); !ok {
			return nil, errors.New("invalid Message.Event")
		} else {
			msg.Event = value
		}
	}

	// payload
	if msg.Kind != MessageTypePush {
		return // invalid message, ignore
	}
	if len(arr) > 5 {
		msg.Payload = arr[5]
	}

	return
}
