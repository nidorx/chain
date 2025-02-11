package pubsub

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/pkg"
	"github.com/segmentio/ksuid"
)

var (
	testAdapter    = &testAdapterStruct{messages: []*testAdapterMessage{}, subscriptions: map[string]bool{}}
	remoteId       = ksuid.New()
	remoteIdBytes  = remoteId.Bytes()
	remoteIdString = remoteId.String()
)

func init() {
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}
}

func Test_PubSub_Broadcast_Dispatcher(t *testing.T) {

	topic := "user:123"
	message := []byte("Message 1")

	testClearPubsub()

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)
	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 10)

	received := dispatcher.pop()
	if received == nil {
		t.Errorf("dispatcher did not receive the message")
	}

	expected := &testDispatcherMessage{
		topic:   topic,
		message: message,
		from:    Self(),
	}

	if !reflect.DeepEqual(received, expected) {
		t.Errorf("Invalid response\n   actual: %v\n expected: %v", received, expected)
	}
}

func Test_PubSub_Dispatcher_Remote(t *testing.T) {

	topic := "user:123"
	message := []byte(`[{"id":1}, {"id":2}, {"id":3}, {"id":4}, {"id":5}]`)

	testClearPubsub()
	testAdapter.clear()

	testAsRemote(func() {
		if err := Broadcast(topic, message); err != nil {
			t.Fatal(err)
		}
	})
	remoteMessage := testAdapter.pop()
	<-time.After(time.Millisecond * 10)

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	Dispatch(remoteMessage.topic, remoteMessage.message)

	<-time.After(time.Millisecond * 10)

	received := dispatcher.pop()
	if received == nil {
		t.Errorf("dispatcher did not receive the message")
	}

	expected := &testDispatcherMessage{
		topic:   topic,
		message: message,
		from:    remoteIdString,
	}

	if !reflect.DeepEqual(received, expected) {
		t.Errorf("Invalid response\n   actual: %v\n expected: %v", received, expected)
	}
}

func Test_PubSub_Direct_Broadcast(t *testing.T) {

	topic := "user:123"
	message := []byte(`[{"id":1}, {"id":2}, {"id":3}, {"id":4}, {"id":5}]`)

	testClearPubsub()
	testAdapter.clear()

	destId := selfIdString

	testAsRemote(func() {
		if err := DirectBroadcast(destId, topic, message); err != nil {
			t.Fatal(err)
		}
	})
	remoteMessage := testAdapter.pop()
	if remoteMessage == nil {
		t.Fatal("adapter did not receive the message")
	}
	<-time.After(time.Millisecond * 10)

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	Dispatch(remoteMessage.topic, remoteMessage.message)

	<-time.After(time.Millisecond * 10)

	received := dispatcher.pop()
	if received == nil {
		t.Errorf("dispatcher did not receive the message")
	}

	expected := &testDispatcherMessage{
		topic:   topic,
		message: message,
		from:    remoteIdString,
	}

	if !reflect.DeepEqual(received, expected) {
		t.Errorf("Invalid response\n   actual: %v\n expected: %v", received, expected)
	}
}

func testAsRemote(fn func()) {
	oSelfKSUID := selfId
	oSelfBytes := selfIdBytes
	oSelfString := selfIdString

	defer func() {
		selfId = oSelfKSUID
		selfIdBytes = oSelfBytes
		selfIdString = oSelfString
	}()

	selfId = remoteId
	selfIdBytes = remoteIdBytes
	selfIdString = remoteIdString

	fn()
}

func testClearPubsub() {
	p.subscriptions = &pkg.WildcardStore[*subscription]{}
	p.unsubscribeTimers = map[string]*time.Timer{}

	SetAdapters([]AdapterConfig{{
		Adapter:            testAdapter,
		Topics:             []string{"*"},
		RawMessage:         false,
		DisableCompression: false,
		DisableEncryption:  false,
	}})
}

var testPayloads = []struct {
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

type testDispatcherMessage struct {
	topic   string
	message any
	from    string
}

type testDispatcherStruct struct {
	messages []*testDispatcherMessage
	mutex    sync.Mutex
}

func (d *testDispatcherStruct) Dispatch(topic string, message []byte, from string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.messages = append(d.messages, &testDispatcherMessage{topic, message, from})
}

func (d *testDispatcherStruct) clear() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.messages = []*testDispatcherMessage{}
}

func (d *testDispatcherStruct) pop() *testDispatcherMessage {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if len(d.messages) == 0 {
		return nil
	}
	out := d.messages[len(d.messages)-1]
	d.messages = d.messages[:len(d.messages)-1]
	return out
}

type testAdapterMessage struct {
	topic   string
	message []byte
	opts    map[string]any
}

type testAdapterStruct struct {
	subscriptions map[string]bool
	messages      []*testAdapterMessage
	mutex         sync.RWMutex
}

func (a *testAdapterStruct) Name() string {
	return "test"
}

func (a *testAdapterStruct) Broadcast(topic string, message []byte, opts map[string]any) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.messages = append(a.messages, &testAdapterMessage{topic, message, opts})
	return nil
}

func (a *testAdapterStruct) Subscribe(topic string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.subscriptions[topic] = true
}

func (a *testAdapterStruct) Unsubscribe(topic string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.subscriptions, topic)
}

func (a *testAdapterStruct) subscribed(topic string) bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	_, is := a.subscriptions[topic]
	return is
}

func (a *testAdapterStruct) clear() {
	a.clearMessages()
	a.clearSubscriptions()
}

func (a *testAdapterStruct) clearSubscriptions() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.subscriptions = map[string]bool{}
}

func (a *testAdapterStruct) clearMessages() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.messages = []*testAdapterMessage{}
}

func (a *testAdapterStruct) pop() *testAdapterMessage {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if len(a.messages) == 0 {
		return nil
	}
	out := a.messages[len(a.messages)-1]
	a.messages = a.messages[:len(a.messages)-1]
	return out
}
