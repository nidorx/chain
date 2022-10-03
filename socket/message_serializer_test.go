package socket

import (
	"reflect"
	"strings"
	"testing"
)

//func Test_MessageSerializer_Encode(t *testing.T) {
//
//}

func Test_Socket_MessageSerializer_Decode(t *testing.T) {

	serializer := &MessageSerializer{}

	tests := []struct {
		input    string
		error    string
		expected Message
	}{
		// Push 		= [kind, joinRef, ref, topic, event, payload]
		{`0,2,3,"room:1234","stx_join",{"param1":"foo"}`, "", Message{JoinRef: 2, Ref: 3, Topic: "room:1234", Event: "stx_join", Payload: map[string]any{"param1": "foo"}}},
		{`0,2,3,"room:1234","stx_join"`, "", Message{JoinRef: 2, Ref: 3, Topic: "room:1234", Event: "stx_join"}},
		{`0,2,3,"","stx_join",{"param1":"foo"}`, "", Message{JoinRef: 2, Ref: 3, Event: "stx_join", Payload: map[string]any{"param1": "foo"}}},
		{`0,2,3,"room:1234","",{"param1":"foo"}`, "", Message{JoinRef: 2, Ref: 3, Topic: "room:1234", Payload: map[string]any{"param1": "foo"}}},
		{`0,2,3,"","",{"param1":"foo","param2":"\"a\" \\{}}}"}`, "", Message{JoinRef: 2, Ref: 3, Payload: map[string]any{"param1": "foo", "param2": `"a" \{}}}`}}},
		{`0,2,4,"room:1234","stx_leave",{}`, "", Message{JoinRef: 2, Ref: 4, Topic: "room:1234", Event: "stx_leave", Payload: map[string]any{}}},
		{`0,2,4,"","",{}`, "", Message{JoinRef: 2, Ref: 4, Payload: map[string]any{}}},
		{`0,2,4,"","",[]`, "", Message{JoinRef: 2, Ref: 4, Payload: []any{}}},
		{`0,2,4,"","",[{"param1":"foo"}]`, "", Message{JoinRef: 2, Ref: 4, Payload: []any{map[string]any{"param1": "foo"}}}},
		{`0,2,4,"","",1`, "", Message{JoinRef: 2, Ref: 4, Payload: float64(1)}},
		{`0,2,4,"","","string"`, "", Message{JoinRef: 2, Ref: 4, Payload: "string"}},
		{`0,2,4,"",""`, "", Message{JoinRef: 2, Ref: 4}},
		// Reply 		= [kind, joinRef, ref, topic,        payload]
		// Broadcast 	= [kind,               topic, event, payload]
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := &Message{}
			decoded, err := serializer.Decode([]byte(tt.input), d)
			if err == nil && tt.error != "" {
				t.Errorf("Decode() failed: Invalid Error\n   actual: nil\n expected: %v", tt.error)
			} else if err != nil && tt.error == "" {
				t.Errorf("Decode() failed: Invalid Error\n   actual: %v\n expected: nil", err)
			} else if err != nil && tt.error != "" && !strings.HasPrefix(err.Error(), tt.error) {
				t.Errorf("Decode() failed: Invalid Error\n   actual: %v\n expected: %v", err, tt.error)
			} else if d != decoded {
				t.Errorf("Decode() failed: Invalid Reference\n   actual: %v\n expected: %v", decoded, d)
			} else {
				e := tt.expected
				if e.Kind != d.Kind {
					t.Errorf("Decode() failed: Invalid Kind\n   actual: %v\n expected: %v", d.Kind, e.Kind)
				} else if e.Topic != d.Topic {
					t.Errorf("Decode() failed: Invalid Topic\n   actual: %v\n expected: %v", d.Topic, e.Topic)
				} else if e.Event != d.Event {
					t.Errorf("Decode() failed: Invalid Event\n   actual: %v\n expected: %v", d.Event, e.Event)
				} else if e.Ref != d.Ref {
					t.Errorf("Decode() failed: Invalid Ref\n   actual: %v\n expected: %v", d.Ref, e.Ref)
				} else if e.JoinRef != d.JoinRef {
					t.Errorf("Decode() failed: Invalid Ref\n   actual: %v\n expected: %v", d.JoinRef, e.JoinRef)
				} else if !reflect.DeepEqual(e.Payload, d.Payload) {
					t.Errorf("Decode() failed: Invalid Payload\n   actual: %v\n expected: %v", d.Payload, e.Payload)
				}
			}
		})
	}
}
