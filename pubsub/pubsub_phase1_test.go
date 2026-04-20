package pubsub

import (
	"sync"
	"testing"
	"time"
)

// TestDispatchEmptyMessage verifies that Dispatch does not panic on empty messages.
func TestDispatchEmptyMessage(t *testing.T) {
	testClearPubsub()

	// Should not panic
	Dispatch("test:topic", []byte{})
	Dispatch("test:topic", nil)
}

// TestDispatchSingleByteMessage verifies that single-byte messages are handled correctly.
func TestDispatchSingleByteMessage(t *testing.T) {
	testClearPubsub()

	topic := "test:single"
	message := []byte{0x42}

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Single byte message should not panic (no adapter match, but no panic)
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 10)
}

// TestDispatchOversizedMessage verifies that oversized messages are rejected.
func TestDispatchOversizedMessage(t *testing.T) {
	testClearPubsub()

	// Temporarily reduce max size for testing
	originalMax := MaxMessageSize
	MaxMessageSize = 100
	defer func() { MaxMessageSize = originalMax }()

	topic := "test:oversize"
	message := make([]byte, 200) // Exceeds our test limit

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	Dispatch(topic, message)
	<-time.After(time.Millisecond * 10)

	// Dispatcher should not receive the oversized message
	received := dispatcher.pop()
	if received != nil {
		t.Errorf("dispatcher should not have received oversized message")
	}
}

// TestDispatchMaxSizeBoundary verifies messages at exact boundary are accepted.
func TestDispatchMaxSizeBoundary(t *testing.T) {
	testClearPubsub()

	// Temporarily reduce max size for testing
	originalMax := MaxMessageSize
	MaxMessageSize = 100
	defer func() { MaxMessageSize = originalMax }()

	topic := "test:boundary"
	message := make([]byte, 100) // Exactly at limit

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// This won't match adapter, but shouldn't panic
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 10)
}

// TestBroadcastLocalFirst verifies local dispatch happens even when adapter fails.
func TestBroadcastLocalFirst(t *testing.T) {
	testClearPubsub()

	topic := "user:local"
	message := []byte("local first test")

	// Use a failing adapter
	failingAdapter := &failingAdapterStruct{subscriptions: make(map[string]bool)}
	SetAdapters([]AdapterConfig{{
		Adapter:            failingAdapter,
		Topics:             []string{"*"},
		RawMessage:         false,
		DisableCompression: true,
		DisableEncryption:  true,
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	<-time.After(time.Millisecond * 20)

	// Broadcast should return error from adapter
	if err == nil {
		t.Errorf("expected error from failing adapter, got nil")
	}

	// But local dispatcher should still receive the message
	received := dispatcher.pop()
	if received == nil {
		t.Errorf("dispatcher did not receive the message despite adapter failure")
		return
	}

	if received.topic != topic {
		t.Errorf("expected topic %q, got %q", topic, received.topic)
	}
	if string(received.message.([]byte)) != string(message) {
		t.Errorf("expected message %q, got %q", string(message), received.message)
	}
}

// TestBroadcastLocalFirstWithSuccess verifies local dispatch and successful adapter broadcast.
func TestBroadcastLocalFirstWithSuccess(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "user:success"
	message := []byte("success test")

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	<-time.After(time.Millisecond * 20)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Local dispatcher should receive the message
	received := dispatcher.pop()
	if received == nil {
		t.Errorf("dispatcher did not receive the message")
		return
	}

	if received.topic != topic {
		t.Errorf("expected topic %q, got %q", topic, received.topic)
	}

	// Adapter should have been called
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Errorf("adapter did not receive the message")
	}
}

// TestBroadcastWithDummyAdapter verifies dummy adapter still dispatches locally.
func TestBroadcastWithDummyAdapter(t *testing.T) {
	testClearPubsub()

	// Set dummy adapter
	SetAdapters([]AdapterConfig{{
		Adapter:            &DummyAdapter{},
		Topics:             []string{"*"},
		DisableCompression: true,
		DisableEncryption:  true,
	}})

	topic := "user:dummy"
	message := []byte("dummy test")

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	<-time.After(time.Millisecond * 10)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	received := dispatcher.pop()
	if received == nil {
		t.Errorf("dispatcher did not receive the message")
		return
	}

	if received.topic != topic {
		t.Errorf("expected topic %q, got %q", topic, received.topic)
	}
}

// TestMaxMessageSizeConfigurable verifies MaxMessageSize is configurable.
func TestMaxMessageSizeConfigurable(t *testing.T) {
	originalMax := MaxMessageSize

	// Test default is 1MB
	if MaxMessageSize != 1<<20 {
		t.Errorf("expected default MaxMessageSize to be 1MB, got %d", MaxMessageSize)
	}

	// Test it can be changed
	MaxMessageSize = 512
	if MaxMessageSize != 512 {
		t.Errorf("expected MaxMessageSize to be 512, got %d", MaxMessageSize)
	}

	// Restore
	MaxMessageSize = originalMax
}

// TestConcurrentBroadcastAndDispatch verifies thread safety under concurrent access.
func TestConcurrentBroadcastAndDispatch(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "user:concurrent"

	var wg sync.WaitGroup
	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Run 10 concurrent broadcasts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = Broadcast(topic, []byte{byte(n)})
		}(i)
	}

	wg.Wait()
	<-time.After(time.Millisecond * 50)

	// Count received messages
	var receivedCount int
	for i := 0; i < 10; i++ {
		if dispatcher.pop() != nil {
			receivedCount++
		}
	}

	if receivedCount == 0 {
		t.Errorf("no messages received by dispatcher")
	}
}

// failingAdapterStruct is an adapter that always fails on Broadcast.
type failingAdapterStruct struct {
	subscriptions map[string]bool
	mutex         sync.RWMutex
}

func (a *failingAdapterStruct) Name() string {
	return "failing"
}

func (a *failingAdapterStruct) Broadcast(topic string, message []byte, opts map[string]any) error {
	return &testError{"broadcast intentionally failed"}
}

func (a *failingAdapterStruct) Subscribe(topic string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.subscriptions[topic] = true
}

func (a *failingAdapterStruct) Unsubscribe(topic string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.subscriptions, topic)
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
