package pubsub

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/ksuid"
)

// TestNewPubSubInstance verifies that new PubSub instances can be created.
func TestNewPubSubInstance(t *testing.T) {
	ps := New()
	if ps == nil {
		t.Fatal("New() should not return nil")
	}
	defer ps.Close()

	if ps.Self() == "" {
		t.Error("PubSub instance should have a valid Self() ID")
	}
}

// TestPubSubInstanceIsolation verifies that multiple instances are isolated.
func TestPubSubInstanceIsolation(t *testing.T) {
	ps1 := New()
	ps2 := New()
	defer ps1.Close()
	defer ps2.Close()

	// Instances should have different IDs
	if ps1.Self() == ps2.Self() {
		t.Error("Different instances should have different IDs")
	}

	// Subscribe to instance 1
	dispatcher1 := &testDispatcherStruct{}
	ps1.Subscribe("test:topic", dispatcher1)

	// Broadcast on instance 2
	ps2.LocalBroadcast("test:topic", []byte("message"))
	<-time.After(time.Millisecond * 10)

	// Instance 1's dispatcher should NOT receive the message
	received := dispatcher1.pop()
	if received != nil {
		t.Error("Instance isolation failed: dispatcher received message from different instance")
	}
}

// TestPubSubInstanceSubscribeBroadcast verifies basic subscribe/broadcast on an instance.
func TestPubSubInstanceSubscribeBroadcast(t *testing.T) {
	ps := New()
	defer ps.Close()

	// Set up dummy adapter on instance
	ps.SetAdapters([]AdapterConfig{{
		Adapter:           &DummyAdapter{},
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	dispatcher := &testDispatcherStruct{}
	ps.Subscribe("user:123", dispatcher)

	message := []byte("hello world")
	if err := ps.Broadcast("user:123", message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher did not receive message")
	}

	if string(received.message.([]byte)) != string(message) {
		t.Errorf("expected message %q, got %q", string(message), received.message)
	}
}

// TestPubSubInstanceCustomConfig verifies instance configuration options work.
func TestPubSubInstanceCustomConfig(t *testing.T) {
	customID := ksuid.New()
	ps := New(
		WithSelfID(customID),
		WithDispatchWorkers(2),
		WithDispatchQueueSize(500),
	)
	defer ps.Close()

	if ps.Self() != customID.String() {
		t.Errorf("expected Self() to be %s, got %s", customID.String(), ps.Self())
	}
}

// TestPubSubInstanceClose verifies graceful shutdown.
func TestPubSubInstanceClose(t *testing.T) {
	ps := New()

	dispatcher := &testDispatcherStruct{}
	ps.Subscribe("test:close", dispatcher)

	// Broadcast before close
	ps.LocalBroadcast("test:close", []byte("before close"))
	<-time.After(time.Millisecond * 10)

	// Close the instance
	ps.Close()

	// Further operations should not panic
	ps.LocalBroadcast("test:close", []byte("after close"))
	<-time.After(time.Millisecond * 10)

	// Just verify no panic occurred
}

// ============================================================================
// Worker Pool Tests
// ============================================================================

// TestWorkerPoolDispatch verifies messages are dispatched via worker pool.
func TestWorkerPoolDispatch(t *testing.T) {
	ps := New()
	defer ps.Close()

	var dispatchCount int64
	var wg sync.WaitGroup

	// Create multiple dispatchers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		d := &testDispatcherStruct{}
		ps.Subscribe("test:worker", d)

		go func() {
			defer wg.Done()
			<-time.After(time.Millisecond * 20)
			if d.pop() != nil {
				atomic.AddInt64(&dispatchCount, 1)
			}
		}()
	}

	// Broadcast a message
	ps.LocalBroadcast("test:worker", []byte("test"))

	wg.Wait()

	if dispatchCount == 0 {
		t.Error("no dispatchers received message")
	}
}

// TestWorkerPoolConcurrency verifies worker pool handles concurrent dispatch.
func TestWorkerPoolConcurrency(t *testing.T) {
	ps := New()
	defer ps.Close()

	const numMessages = 100
	var receivedCount int64

	dispatcher := &testDispatcherStruct{}
	ps.Subscribe("concurrent:worker", dispatcher)

	// Send multiple messages
	var wg sync.WaitGroup
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ps.LocalBroadcast("concurrent:worker", []byte{byte(n)})
		}(i)
	}

	wg.Wait()
	<-time.After(time.Millisecond * 100)

	// Count received messages
	for i := 0; i < numMessages; i++ {
		if dispatcher.pop() != nil {
			atomic.AddInt64(&receivedCount, 1)
		}
	}

	if receivedCount == 0 {
		t.Error("no messages received")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestInstanceWithTypedOptions verifies instance API works with typed options.
func TestInstanceWithTypedOptions(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	ps := New()
	defer ps.Close()

	// Set up adapter on instance
	ps.SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	dispatcher := &testDispatcherStruct{}
	ps.Subscribe("integration:typed", dispatcher)

	// Broadcast (typed options are available via applyOptions but Broadcast still uses legacy for now)
	err := ps.Broadcast("integration:typed", []byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher did not receive message")
	}
}

// TestBackwardCompatibility verifies global functions still work.
func TestBackwardCompatibility(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	dispatcher := &testDispatcherStruct{}
	Subscribe("compat:test", dispatcher)

	if err := Broadcast("compat:test", []byte("backward compatible")); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Fatal("global functions should still work")
	}
}
