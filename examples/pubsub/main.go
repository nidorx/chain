package main

import (
	"fmt"
	"github.com/syntax-framework/chain/pubsub"
	"time"
)

type MyDispatcher struct {
}

func (d *MyDispatcher) Dispatch(topic string, message any) {
	println(fmt.Sprintf("New Message. Topic: %s, Content: %v", topic, message))
}

func main() {

	dispatcher := &MyDispatcher{}

	pubsub.Subscribe("user:123", dispatcher)

	pubsub.Broadcast("user:123", map[string]any{
		"Event": "user_update",
		"Payload": map[string]any{
			"Id":   6,
			"Name": "Gabriel",
		},
	})

	pubsub.Broadcast("user:123", "Message 2")

	// await
	<-time.After(time.Second)

	pubsub.Unsubscribe("user:123", dispatcher)

	pubsub.Broadcast("user:123", "Message Ignored")
}
