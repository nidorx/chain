package socket

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
)

type MessageSerializer struct{}

func (s *MessageSerializer) Encode(v any) (data []byte, err error) {
	var msg *Message
	var valid bool
	if msg, valid = v.(*Message); !valid {
		err = errors.New("can only serialize *Message")
		return
	}

	// Push 		= [kind, joinRef, ref,   topic, event, payload]
	// Reply 		= [kind, joinRef, ref, status,        payload]
	// Broadcast 	= [kind,                topic, event, payload]
	buf := &bytes.Buffer{}
	buf.Write([]byte(strconv.Itoa(int(msg.Kind))))
	if msg.Kind != MessageTypeBroadcast {
		buf.WriteRune(',')
		buf.Write([]byte(strconv.Itoa(msg.JoinRef)))
		buf.WriteRune(',')
		buf.Write([]byte(strconv.Itoa(msg.Ref)))
	}

	if msg.Kind == MessageTypeReply {
		buf.WriteRune(',')
		buf.Write([]byte(strconv.Itoa(msg.Status)))
	} else {
		if msg.Topic == "" {
			buf.WriteString(`,""`)
		} else {
			data, err = json.Marshal(msg.Topic)
			if err != nil {
				return
			}
			buf.WriteRune(',')
			buf.Write(data)
		}
	}

	if msg.Kind != MessageTypeReply {
		if msg.Event == "" {
			buf.WriteString(`,""`)
		} else {
			data, err = json.Marshal(msg.Event)
			if err != nil {
				return
			}
			buf.WriteRune(',')
			buf.Write(data)
		}
	}

	if msg.Payload != nil {
		data, err = json.Marshal(msg.Payload)
		if err != nil {
			return
		}
		buf.WriteRune(',')
		buf.Write(data)
	}

	data = buf.Bytes()

	return
}

func (s *MessageSerializer) Decode(data []byte, v any) (out any, err error) {
	var valid bool
	var msg *Message
	if msg, valid = v.(*Message); !valid {
		err = errors.New("can only deserialize *Message")
		return
	}
	out = msg

	var (
		auxInt     int
		fieldIdx   = 0
		fieldStart = 0
		fieldEnd   = 0
		brackets   = 0
		inQuote    = false
	)
	for i, b := range data {
		if b == '"' {
			if i > 0 && data[i-1] != '\\' {
				inQuote = !inQuote
			} else if i == 0 && b == '"' {
				inQuote = true
			}
		} else if !inQuote && b == '{' {
			brackets++
		} else if !inQuote && b == '}' {
			brackets--
		}
		if (!inQuote && brackets == 0 && b == ',') || i == len(data)-1 {
			fieldEnd = i
			if i == len(data)-1 {
				fieldEnd++
			}

			// Push 		= kind, joinRef, ref,  topic, event, payload
			// Reply 		= kind, joinRef, ref, status,        payload
			// Broadcast 	= kind,                topic, event, payload
			switch fieldIdx {
			case 0: // kind
				auxInt, err = strconv.Atoi(string(data[fieldStart:fieldEnd]))
				if err != nil {
					err = errors.New("invalid Message.Kind. msg:" + err.Error())
					return
				}
				msg.Kind = MessageType(auxInt)
			case 1: // joinRef | topic
				if msg.Kind == MessageTypeBroadcast {
					// topic
					msg.Topic = string(data[fieldStart+1 : fieldEnd-1])
				} else {
					// joinRef
					auxInt, err = strconv.Atoi(string(data[fieldStart:fieldEnd]))
					if err != nil {
						err = errors.New("invalid Message.JoinRef. msg:" + err.Error())
						return
					}
					msg.JoinRef = auxInt
				}
			case 2: // ref | event
				if msg.Kind == MessageTypeBroadcast {
					// event
					msg.Event = string(data[fieldStart:fieldEnd])
				} else {
					// ref
					auxInt, err = strconv.Atoi(string(data[fieldStart:fieldEnd]))
					if err != nil {
						err = errors.New("invalid Message.Ref. msg:" + err.Error())
						return
					}
					msg.Ref = auxInt
				}
			case 3: // topic | payload | status
				if msg.Kind == MessageTypeReply {
					// status
					auxInt, err = strconv.Atoi(string(data[fieldStart:fieldEnd]))
					if err != nil {
						err = errors.New("invalid Message.Status. msg:" + err.Error())
						return
					}
					msg.Status = auxInt
				} else if msg.Kind == MessageTypeBroadcast {
					// payload
					var payload any
					if payload, err = s.decodePayload(data[fieldStart:fieldEnd]); err != nil {
						err = errors.New("invalid Message.Payload. msg:" + err.Error())
						return
					}
					msg.Payload = payload
				} else {
					// topic
					msg.Topic = string(data[fieldStart+1 : fieldEnd-1])
				}
			case 4: // event | payload
				if msg.Kind == MessageTypeBroadcast {
					return // invalid message, ignore
				}

				if msg.Kind == MessageTypeReply {
					var payload any
					if payload, err = s.decodePayload(data[fieldStart:fieldEnd]); err != nil {
						err = errors.New("invalid Message.Payload. msg:" + err.Error())
						return
					}
					msg.Payload = payload
				} else {
					msg.Event = string(data[fieldStart+1 : fieldEnd-1])
				}
			case 5: // payload
				if msg.Kind != MessageTypePush {
					return // invalid message, ignore
				}
				var payload any
				if payload, err = s.decodePayload(data[fieldStart:fieldEnd]); err != nil {
					err = errors.New("invalid Message.Payload. msg:" + err.Error())
					return nil, err
				}
				msg.Payload = payload
			}

			fieldIdx++
			fieldStart = i + 1
		}
	}
	return
}

func (s *MessageSerializer) decodePayload(data []byte) (out any, err error) {
	if data[0] == '{' {

	}
	switch data[0] {
	case '{':
		out = map[string]any{}
	case '[':
		out = []any{}
	case '"':
		out = ""
	case 't': // true
		out = true
		return
	case 'f': // false
		out = false
		return
	case 'n': // null
		out = nil
		return
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-': // number:
		var f float64
		if f, err = strconv.ParseFloat(string(data), 64); err != nil {
			err = errors.New("invalid number. msg" + err.Error())
			return
		}
		out = f
		return
	}

	err = json.Unmarshal(data, &out)
	return
}
