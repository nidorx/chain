package pubsub

import (
	"fmt"
	"github.com/syntax-framework/chain"
	"testing"
	"time"
)

var payloads = []struct {
	content string
}{
	{"test"},
	{"message"},
	{`{"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}`},
	{`[{"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]},{"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]},{"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}]`},
	{`[
		{"Name": "Platypus", "Order": "Monotremata"}, 
		{"Name": "Quoll",    "Order": "Dasyuromorphia"}
	]`},
	{`
		{"Name": "Ed", "Text": "Knock knock."}
		{"Name": "Sam", "Text": "Who's there?"}
		{"Name": "Ed", "Text": "Go fmt."}
		{"Name": "Sam", "Text": "Go fmt who?"}
		{"Name": "Ed", "Text": "Go fmt yourself!"}
	`},
}

type dispatcherT struct {
}

func (d *dispatcherT) Dispatch(topic string, message any, from string) {
	println(fmt.Sprintf("New Message. Topic: %s, Content: %s", topic, message))
}

type adapterT struct {
}

func (a *adapterT) Name() string {
	return "test"
}

func (a *adapterT) Broadcast(topic string, message []byte, opts map[string]any) error {
	// do nothing
	return nil
}

func (a *adapterT) Subscribe(topic string) {
	// do nothing
}

func (a *adapterT) Unsubscribe(topic string) {
	// do nothing
}

func Test_PubSub_Broadcast(t *testing.T) {
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}

	SetAdapters([]AdapterConfig{{
		Adapter:            &adapterT{},
		Topics:             []string{"*"},
		RawMessage:         false,
		DisableCompression: false,
		DisableEncryption:  false,
	}})

	dispatcher := &dispatcherT{}

	Subscribe("user:123", dispatcher)

	Broadcast("user:123", []byte("Message 1"))

	<-time.After(time.Second)
}
