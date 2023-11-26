package socket

import (
	"context"
	"errors"
	"github.com/nidorx/chain"
	"sync"
	"testing"
	"time"
)

type transportT struct {
	handler   *Handler
	info      *Session
	OnMessage func(bytes []byte, message *Message, err error)
	Errors    []error
	Messages  []*Message
	mutex     sync.Mutex
	ctx       context.Context
	kill      context.CancelFunc
}

func (t *transportT) Configure(h *Handler, r *chain.Router, endpoint string) {
	t.handler = h
}

func (t *transportT) putMessage(bytes []byte) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	message := newMessageAny()
	_, err := t.handler.Serializer.Decode(bytes, message)
	if err != nil {
		t.Errors = append(t.Errors, err)
	} else {
		t.Messages = append(t.Messages, message)
	}
}

func (t *transportT) Connect(params map[string]string) (*Session, error) {
	session, err := t.handler.Connect("/test", params)
	t.info = session
	if err == nil {
		ctx, kill := context.WithCancel(context.Background())
		t.ctx = ctx
		t.kill = kill
		go func() {
			defer session.ScheduleShutdown(time.Millisecond * 10)
			// trap the request under loop forever
			for {
				select {
				case <-t.ctx.Done():
					return
				case msg := <-session.messages:
					t.putMessage(msg)
				}
			}
		}()
	}
	return session, err
}

func (t *transportT) Close() {
	t.kill()
}

func (t *transportT) SendMessage(message *Message) {
	bytes, _ := t.handler.Serializer.Encode(message)
	t.handler.Dispatch(bytes, t.info)
}

func (t *transportT) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Errors = []error{}
	t.Messages = []*Message{}
}

func (t *transportT) PopMessage() *Message {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if len(t.Messages) == 0 {
		return nil
	}
	out := t.Messages[len(t.Messages)-1]
	t.Messages = t.Messages[:len(t.Messages)-1]
	return out
}

func (t *transportT) PopError() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.Errors[0]
}

func Test_Socket(t *testing.T) {

	errParamValidFalse := errors.New("info.Param.Valid == false")

	signature := ""

	transport := &transportT{}

	router := chain.New()
	handler := &Handler{
		Transports: []Transport{transport},
		Channels: []*Channel{
			NewChannel("chat:*", func(channel *Channel) {

				channel.Join("chat:lobby", func(payload any, socket *Socket) (reply any, err error) {
					signature = signature + "JOIN "
					signature = signature + socket.Params["connect.param"] + " "
					if m, v := payload.(map[string]any); v {
						signature = signature + m["id"].(string) + " "
					}

					socket.Set("Join.Value", "JOIN_VAL")

					return
				})

				channel.HandleIn("event", func(event string, payload any, socket *Socket) (reply any, err error) {
					signature = signature + "IN "
					signature = signature + socket.Get("Join.Value").(string) + " "
					if m, v := payload.(map[string]any); v {
						signature = signature + m["payload"].(string) + " "
					}

					err = socket.Push("eventResponse", map[string]any{"response.value": "RESPONSE_VAL"})
					reply = map[string]any{"reply.value": "REPLY_VAL"}

					return
				})

				channel.Leave("chat:lobby", func(socket *Socket, reason LeaveReason) {
					signature = signature + "LEAVE "
					signature = signature + socket.Get("Join.Value").(string) + " "
				})
			}),
		},
		OnConfig: func(handler *Handler, router *chain.Router, endpoint string) error {
			handler.Options["options.valid"] = true
			signature = signature + "OnConfig "
			return nil
		},
		OnConnect: func(info *Session) error {
			if info.Params["valid"] == "false" {
				return errParamValidFalse
			}

			signature = signature + "OnConnect "
			return nil
		},
	}
	router.Configure("/socket", handler)

	// TESTS

	// Connect error
	_, err := transport.Connect(map[string]string{"valid": "false"})
	if err != errParamValidFalse {
		t.Errorf("Connect() failed: invalid error\n   actual: %v\n expected: %v", err, errParamValidFalse)
		return
	}

	// Connect success
	if _, err = transport.Connect(map[string]string{"connect.param": "CONN_PARAM"}); err != nil {
		t.Errorf("Connect() failed: unexpected error: %v", err)
		return
	}

	var request *Message
	var response *Message
	var payload map[string]any
	var ok bool

	// Channel JOIN
	request = newMessage(MessageTypePush, "chat:lobby", "stx_join", map[string]any{"id": "USER1"})
	request.Ref = 1
	request.JoinRef = 1
	transport.SendMessage(request)
	time.Sleep(time.Second)

	response = transport.PopMessage()
	if response.Kind != MessageTypeReply {
		t.Errorf("Join() failed: invalid reply Kind\n   actual: %v\n expected: %v", response.Kind, MessageTypeReply)
	}
	if response.Ref != request.Ref {
		t.Errorf("Join() failed: invalid reply Ref\n   actual: %v\n expected: %v", response.Ref, request.Ref)
	}
	if response.JoinRef != request.JoinRef {
		t.Errorf("Join() failed: invalid reply JoinRef\n   actual: %v\n expected: %v", response.JoinRef, request.JoinRef)
	}

	// Channel EVENT
	request = newMessage(MessageTypePush, "chat:lobby", "event", map[string]any{"payload": "EVT_VAL"})
	request.Ref = 2
	request.JoinRef = 1
	transport.SendMessage(request)
	time.Sleep(time.Second)

	// reply
	response = transport.PopMessage()
	if response.Kind != MessageTypeReply {
		t.Errorf("Push() failed: invalid reply Kind\n   actual: %v\n expected: %v", response.Kind, MessageTypeReply)
	}
	if response.Ref != request.Ref {
		t.Errorf("Push() failed: invalid reply Ref\n   actual: %v\n expected: %v", response.Ref, request.Ref)
	}
	if response.JoinRef != request.JoinRef {
		t.Errorf("Push() failed: invalid reply JoinRef\n   actual: %v\n expected: %v", response.JoinRef, request.JoinRef)
	}

	if payload, ok = response.Payload.(map[string]any); !ok {
		t.Errorf("Push() failed: invalid reply Payload\n   actual: %v\n expected: %v", response.Payload, "map[string]any")
	}
	if payload["reply.value"] != "REPLY_VAL" {
		t.Errorf("Push() failed: invalid reply Payload\n   actual: %v\n expected: %v", payload["reply.value"], "REPLY_VAL")
	}

	// eventResponse
	response = transport.PopMessage()
	if response.Kind != MessageTypePush {
		t.Errorf("Push() failed: invalid reply Kind\n   actual: %v\n expected: %v", response.Kind, MessageTypePush)
	}

	if response.JoinRef != request.JoinRef {
		t.Errorf("Push() failed: invalid reply JoinRef\n   actual: %v\n expected: %v", response.JoinRef, request.JoinRef)
	}
	if response.Topic != "chat:lobby" {
		t.Errorf("Push() failed: invalid reply Topic\n   actual: %v\n expected: %v", response.Topic, "chat:lobby")
	}
	if response.Event != "eventResponse" {
		t.Errorf("Push() failed: invalid reply Event\n   actual: %v\n expected: %v", response.Event, "eventResponse")
	}

	if payload, ok = response.Payload.(map[string]any); !ok {
		t.Errorf("Push() failed: invalid reply Payload\n   actual: %v\n expected: %v", response.Payload, "map[string]any")
	}
	if payload["response.value"] != "RESPONSE_VAL" {
		t.Errorf("Push() failed: invalid reply Payload\n   actual: %v\n expected: %v", payload["response.value"], "RESPONSE_VAL")
	}

	// Channel LEAVE
	request = newMessage(MessageTypePush, "chat:lobby", "stx_leave", nil)
	request.Ref = 3
	request.JoinRef = 1
	transport.SendMessage(request)
	time.Sleep(time.Second)

	response = transport.PopMessage()
	if response.Kind != MessageTypeReply {
		t.Errorf("Join() failed: invalid reply Kind\n   actual: %v\n expected: %v", response.Kind, MessageTypeReply)
	}
	if response.Ref != request.Ref {
		t.Errorf("Join() failed: invalid reply Ref\n   actual: %v\n expected: %v", response.Ref, request.Ref)
	}
	if response.JoinRef != request.JoinRef {
		t.Errorf("Join() failed: invalid reply JoinRef\n   actual: %v\n expected: %v", response.JoinRef, request.JoinRef)
	}

	expectedSignature := "OnConfig OnConnect JOIN CONN_PARAM USER1 IN JOIN_VAL EVT_VAL LEAVE JOIN_VAL "
	if signature != expectedSignature {
		t.Errorf("invalid signature\n   actual: %v\n expected: %v", signature, expectedSignature)
	}

}
