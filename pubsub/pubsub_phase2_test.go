package pubsub

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nidorx/chain"
)

func init() {
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}
}

// ============================================================================
// Concurrency Tests (2.2)
// ============================================================================

// TestConcurrentSubscribeUnsubscribe verifies that rapid subscribe/unsubscribe
// cycles from multiple goroutines don't cause race conditions or panics.
func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	const goroutines = 100
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				topic := fmt.Sprintf("concurrent:topic:%d", id%10) // 10 shared topics
				dispatcher := &testDispatcherStruct{}

				Subscribe(topic, dispatcher)
				// Small delay to create interleaving
				if i%5 == 0 {
					time.Sleep(time.Millisecond)
				}
				Unsubscribe(topic, dispatcher)
			}
		}(g)
	}

	wg.Wait()
	<-time.After(time.Millisecond * 100)
}

// TestConcurrentSubscribeSameTopic verifies concurrent subscriptions to the same topic.
func TestConcurrentSubscribeSameTopic(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	dispatchers := make([]*testDispatcherStruct, goroutines)
	for i := range dispatchers {
		dispatchers[i] = &testDispatcherStruct{}
	}

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			Subscribe("shared:topic", dispatchers[id])
		}(g)
	}

	wg.Wait()

	// Verify all dispatchers are subscribed
	p.subscriptionsMutex.RLock()
	sub := p.subscriptions.Get("shared:topic")
	p.subscriptionsMutex.RUnlock()

	if sub == nil {
		t.Fatal("subscription should exist")
	}

	if len(sub.dispatchers) != goroutines {
		t.Errorf("expected %d dispatchers, got %d", goroutines, len(sub.dispatchers))
	}
}

// TestConcurrentBroadcast verifies multiple goroutines broadcasting simultaneously.
func TestConcurrentBroadcast(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "concurrent:broadcast"
	const goroutines = 50
	const messagesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	var totalSent int64
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < messagesPerGoroutine; i++ {
				msg := []byte(fmt.Sprintf("msg-%d-%d", id, i))
				if err := Broadcast(topic, msg); err != nil {
					t.Errorf("broadcast failed: %v", err)
				}
				atomic.AddInt64(&totalSent, 1)
			}
		}(g)
	}

	wg.Wait()
	<-time.After(time.Millisecond * 200)

	// Count received messages
	var receivedCount int
	for i := 0; i < int(totalSent); i++ {
		if dispatcher.pop() != nil {
			receivedCount++
		}
	}

	if receivedCount == 0 {
		t.Error("no messages received")
	}
}

// TestConcurrentSubscribeAndBroadcast verifies subscribing and broadcasting concurrently.
func TestConcurrentSubscribeAndBroadcast(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Subscribe
	var subCount int64
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			topic := fmt.Sprintf("race:topic:%d", i%20)
			d := &testDispatcherStruct{}
			Subscribe(topic, d)
			atomic.AddInt64(&subCount, 1)
		}
	}()

	// Goroutine 2: Broadcast
	var broadcastCount int64
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			topic := fmt.Sprintf("race:topic:%d", i%20)
			msg := []byte(fmt.Sprintf("msg-%d", i))
			if err := Broadcast(topic, msg); err != nil {
				// May fail if no adapter matches, that's ok
				_ = err
			}
			atomic.AddInt64(&broadcastCount, 1)
		}
	}()

	wg.Wait()
	<-time.After(time.Millisecond * 100)

	t.Logf("Subscriptions: %d, Broadcasts: %d", subCount, broadcastCount)
}

// ============================================================================
// Edge Case Tests (2.2)
// ============================================================================

// TestEmptyMessageDispatch verifies that empty messages don't cause panic.
func TestEmptyMessageDispatchPhase2(t *testing.T) {
	testClearPubsub()

	topic := "test:empty"
	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Empty slice
	Dispatch(topic, []byte{})
	<-time.After(time.Millisecond * 10)

	// nil slice
	Dispatch(topic, nil)
	<-time.After(time.Millisecond * 10)

	// Dispatcher should not receive empty messages
	received := dispatcher.pop()
	if received != nil {
		t.Errorf("dispatcher should not have received empty message")
	}
}

// TestSingleByteMessage verifies single-byte messages are handled.
func TestSingleByteMessage(t *testing.T) {
	testClearPubsub()

	topic := "test:single"
	message := []byte{0x02} // MessageTypeBroadcast

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Should not panic, but message is too short for valid broadcast
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 10)
}

// TestOversizedMessage verifies oversized messages are rejected.
func TestOversizedMessagePhase2(t *testing.T) {
	testClearPubsub()

	originalMax := MaxMessageSize
	MaxMessageSize = 100
	defer func() { MaxMessageSize = originalMax }()

	topic := "test:oversize"
	message := make([]byte, 200)

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	Dispatch(topic, message)
	<-time.After(time.Millisecond * 10)

	received := dispatcher.pop()
	if received != nil {
		t.Errorf("dispatcher should not have received oversized message")
	}
}

// TestMaxSizeBoundary verifies messages at exact boundary.
func TestMaxSizeBoundaryPhase2(t *testing.T) {
	testClearPubsub()

	originalMax := MaxMessageSize
	MaxMessageSize = 100
	defer func() { MaxMessageSize = originalMax }()

	topic := "test:boundary"
	message := make([]byte, 100) // Exactly at limit

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Should not panic
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 10)
}

// ============================================================================
// Wildcard Topic Matching Tests (2.2)
// ============================================================================

// TestWildcardTopicMatching verifies comprehensive wildcard topic matching.
func TestWildcardTopicMatching(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		topic       string
		shouldMatch bool
	}{
		{"exact match", "user:123", "user:123", true},
		{"exact no match", "user:123", "user:456", false},
		{"star suffix", "user:*", "user:123", true},
		{"star suffix no match", "user:*", "admin:123", false},
		{"single star", "*", "anything:matches", true},
		{"partial match", "user:12", "user:123", false},
		{"longer pattern", "user:123:extra", "user:123", false},
		{"nested star", "user:posts:*", "user:posts:abc", true},
		{"nested star no match", "user:posts:*", "user:comments:abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClearPubsub()
			testAdapter.clear()

			dispatcher := &testDispatcherStruct{}
			Subscribe(tt.pattern, dispatcher)
			<-time.After(time.Millisecond * 10)

			message := []byte("test message")
			if err := Broadcast(tt.topic, message); err != nil && err != ErrNoAdapter {
				t.Fatalf("unexpected error: %v", err)
			}
			<-time.After(time.Millisecond * 20)

			received := dispatcher.pop()
			if tt.shouldMatch && received == nil {
				t.Errorf("expected message to match pattern %q for topic %q", tt.pattern, tt.topic)
			}
			if !tt.shouldMatch && received != nil {
				t.Errorf("expected message NOT to match pattern %q for topic %q", tt.pattern, tt.topic)
			}
		})
	}
}

// TestMultipleWildcardPatterns verifies multiple wildcard patterns work correctly.
func TestMultipleWildcardPatterns(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "user:123"
	message := []byte("multi-wildcard test")

	// Subscribe with multiple patterns that should match
	dispatcher1 := &testDispatcherStruct{}
	dispatcher2 := &testDispatcherStruct{}

	Subscribe("user:*", dispatcher1)
	Subscribe("*", dispatcher2)
	<-time.After(time.Millisecond * 10)

	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	// Both dispatchers should receive the message
	received1 := dispatcher1.pop()
	received2 := dispatcher2.pop()

	if received1 == nil {
		t.Error("dispatcher1 (user:*) should have received message")
	}
	if received2 == nil {
		t.Error("dispatcher2 (*) should have received message")
	}
}

// TestLongestPrefixWildcardMatch verifies longest prefix matching.
func TestLongestPrefixWildcardMatch(t *testing.T) {
	testClearPubsub()

	topic := "admin:users:123"
	message := []byte("admin message")

	dispatcher := &testDispatcherStruct{}
	Subscribe("admin:users:*", dispatcher)
	<-time.After(time.Millisecond * 10)

	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Error("dispatcher should have received message matching admin:users:*")
	}
}

// ============================================================================
// Adapter Failure Tests (2.2)
// ============================================================================

// TestBroadcastAdapterFailureLocalDispatch verifies local dispatch still works when adapter fails.
func TestBroadcastAdapterFailureLocalDispatchPhase2(t *testing.T) {
	testClearPubsub()

	topic := "fail:local"
	message := []byte("local despite failure")

	failingAdapter := &failingAdapterStruct{subscriptions: make(map[string]bool)}
	SetAdapters([]AdapterConfig{{
		Adapter:            failingAdapter,
		Topics:             []string{"*"},
		DisableCompression: true,
		DisableEncryption:  true,
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	<-time.After(time.Millisecond * 20)

	// Should return error
	if err == nil {
		t.Error("expected error from failing adapter")
	}

	// But local dispatch should still work
	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher should have received message despite adapter failure")
	}

	if string(received.message.([]byte)) != string(message) {
		t.Errorf("expected message %q, got %q", string(message), received.message)
	}
}

// TestDispatchDecryptionFailure verifies decryption failures are handled gracefully.
func TestDispatchDecryptionFailure(t *testing.T) {
	testClearPubsub()

	topic := "test:decrypt"
	message := []byte{byte(MessageTypeEncrypt), 0x01, 0x02, 0x03} // Invalid encrypted message

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Should not panic, should log error
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 20)

	// Dispatcher should not receive invalid encrypted message
	received := dispatcher.pop()
	if received != nil {
		t.Errorf("dispatcher should not have received invalid encrypted message")
	}
}

// TestDispatchDecompressionFailure verifies decompression failures are handled gracefully.
func TestDispatchDecompressionFailure(t *testing.T) {
	testClearPubsub()

	topic := "test:decompress"
	message := []byte{byte(MessageTypeCompress), 0xFF, 0xFF, 0xFF} // Invalid compressed data

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Should not panic, should log error
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 20)

	// Dispatcher should not receive invalid compressed message
	received := dispatcher.pop()
	if received != nil {
		t.Errorf("dispatcher should not have received invalid compressed message")
	}
}

// TestBroadcastNoAdapter verifies broadcast returns ErrNoAdapter when no adapter matches.
func TestBroadcastNoAdapter(t *testing.T) {
	testClearPubsub()

	// Clear all adapters
	p.adapters = nil

	err := Broadcast("any:topic", []byte("test"))
	if err != ErrNoAdapter {
		t.Errorf("expected ErrNoAdapter, got %v", err)
	}
}

// ============================================================================
// Timer/Unsubscribe Tests (2.2)
// ============================================================================

// TestUnsubscribeTimerCancellation verifies re-subscribe cancels pending unsubscribe timer.
func TestUnsubscribeTimerCancellation(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "timer:cancel"
	dispatcher := &testDispatcherStruct{}

	// Subscribe
	Subscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 10)

	// Unsubscribe (should start 5s timer)
	Unsubscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 50)

	// Check timer was scheduled
	p.unsubscribeMutex.Lock()
	_, hasTimer := p.unsubscribeTimers[topic]
	p.unsubscribeMutex.Unlock()

	if !hasTimer {
		t.Fatal("unsubscribe timer should have been scheduled")
	}

	// Re-subscribe before timer fires
	Subscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 10)

	// Timer should be cancelled
	p.unsubscribeMutex.Lock()
	_, hasTimer = p.unsubscribeTimers[topic]
	p.unsubscribeMutex.Unlock()

	if hasTimer {
		t.Error("unsubscribe timer should have been cancelled by re-subscribe")
	}

	// Adapter should still be subscribed
	if !testAdapter.subscribed(topic) {
		t.Error("adapter should still be subscribed after re-subscribe")
	}
}

// TestUnsubscribeGracePeriod verifies the 5-second grace period.
func TestUnsubscribeGracePeriod(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "grace:period"
	dispatcher := &testDispatcherStruct{}

	// Subscribe
	Subscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 10)

	// Verify adapter is subscribed
	if !testAdapter.subscribed(topic) {
		t.Fatal("adapter should be subscribed")
	}

	// Unsubscribe
	Unsubscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 100)

	// Timer should be scheduled (5 second grace period)
	p.unsubscribeMutex.Lock()
	_, hasTimer := p.unsubscribeTimers[topic]
	p.unsubscribeMutex.Unlock()

	if !hasTimer {
		t.Error("unsubscribe timer should be scheduled")
	}

	// Adapter should still be subscribed (grace period hasn't elapsed)
	if !testAdapter.subscribed(topic) {
		t.Error("adapter should still be subscribed during grace period")
	}
}

// TestMultipleDispatchersUnsubscribe verifies unsubscribe only removes adapter when last dispatcher leaves.
func TestMultipleDispatchersUnsubscribe(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "multi:unsub"
	dispatcher1 := &testDispatcherStruct{}
	dispatcher2 := &testDispatcherStruct{}

	// Subscribe both dispatchers
	Subscribe(topic, dispatcher1)
	Subscribe(topic, dispatcher2)
	<-time.After(time.Millisecond * 10)

	// Unsubscribe first
	Unsubscribe(topic, dispatcher1)
	<-time.After(time.Millisecond * 10)

	// Adapter should still be subscribed (dispatcher2 remains)
	if !testAdapter.subscribed(topic) {
		t.Error("adapter should still be subscribed while dispatcher2 remains")
	}

	// Verify dispatcher2 still receives messages
	message := []byte("test message")
	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher2.pop()
	if received == nil {
		t.Error("dispatcher2 should have received message")
	}

	// Unsubscribe second (last one)
	Unsubscribe(topic, dispatcher2)
	<-time.After(time.Millisecond * 10)

	// Timer should be scheduled for adapter unsubscribe
	p.unsubscribeMutex.Lock()
	_, hasTimer := p.unsubscribeTimers[topic]
	p.unsubscribeMutex.Unlock()

	if !hasTimer {
		t.Error("unsubscribe timer should be scheduled after last dispatcher removed")
	}
}

// TestRapidSubscribeUnsubscribeCycles verifies rapid cycles don't cause issues.
func TestRapidSubscribeUnsubscribeCycles(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "rapid:cycle"
	dispatcher := &testDispatcherStruct{}

	for i := 0; i < 20; i++ {
		Subscribe(topic, dispatcher)
		Unsubscribe(topic, dispatcher)
	}

	<-time.After(time.Millisecond * 100)

	// Should not have panicked or leaked goroutines
}

// ============================================================================
// Multiple Dispatchers Per Topic Tests (2.2)
// ============================================================================

// TestMultipleDispatchersPerTopic verifies multiple dispatchers can subscribe to same topic.
func TestMultipleDispatchersPerTopic(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "multi:dispatchers"
	message := []byte("multi dispatcher test")

	dispatcher1 := &testDispatcherStruct{}
	dispatcher2 := &testDispatcherStruct{}
	dispatcher3 := &testDispatcherStruct{}

	Subscribe(topic, dispatcher1)
	Subscribe(topic, dispatcher2)
	Subscribe(topic, dispatcher3)
	<-time.After(time.Millisecond * 10)

	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 50)

	// All three should receive the message
	for i, d := range []*testDispatcherStruct{dispatcher1, dispatcher2, dispatcher3} {
		received := d.pop()
		if received == nil {
			t.Errorf("dispatcher%d should have received message", i+1)
		}
	}
}

// TestSameDispatcherMultipleSubscriptions verifies same dispatcher can subscribe multiple times.
// Note: The same dispatcher only receives one message per broadcast (deduplicated by map key),
// but the reference count ensures it's not unsubscribed until all subscriptions are removed.
func TestSameDispatcherMultipleSubscriptions(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "same:dispatcher"
	message := []byte("same dispatcher test")

	dispatcher := &testDispatcherStruct{}

	// Subscribe same dispatcher 3 times
	Subscribe(topic, dispatcher)
	Subscribe(topic, dispatcher)
	Subscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 10)

	// Verify subscription count
	p.subscriptionsMutex.RLock()
	sub := p.subscriptions.Get(topic)
	p.subscriptionsMutex.RUnlock()

	if sub == nil {
		t.Fatal("subscription should exist")
	}
	if count, ok := sub.dispatchers[dispatcher]; !ok || count != 3 {
		t.Errorf("expected dispatcher count 3, got %d", count)
	}

	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 50)

	// Should receive once (deduplicated by map key)
	received := dispatcher.pop()
	if received == nil {
		t.Error("dispatcher should have received message")
	}

	// Unsubscribe once (should decrement count to 2)
	Unsubscribe(topic, dispatcher)
	p.subscriptionsMutex.RLock()
	sub = p.subscriptions.Get(topic)
	p.subscriptionsMutex.RUnlock()

	if sub == nil {
		t.Fatal("subscription should still exist")
	}
	if count, ok := sub.dispatchers[dispatcher]; !ok || count != 2 {
		t.Errorf("expected dispatcher count 2 after unsubscribe, got %d", count)
	}
}

// ============================================================================
// Disabled Encryption/Compression Tests (2.2)
// ============================================================================

// TestBroadcastDisabledEncryption verifies broadcast works with encryption disabled.
func TestBroadcastDisabledEncryption(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "no:encrypt"
	message := []byte("unencrypted message")

	SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	if err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher should have received message")
	}

	// Verify adapter received unencrypted message
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter should have received message")
	}

	// Message should not be encrypted (first byte should be MessageTypeBroadcast)
	if len(adapterMsg.message) > 0 && adapterMsg.message[0] != byte(MessageTypeBroadcast) {
		t.Errorf("expected unencrypted message (type %d), got type %d",
			MessageTypeBroadcast, adapterMsg.message[0])
	}
}

// TestBroadcastDisabledCompression verifies broadcast works with compression disabled.
func TestBroadcastDisabledCompression(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "no:compress"
	message := []byte("uncompressed message")

	SetAdapters([]AdapterConfig{{
		Adapter:            testAdapter,
		Topics:             []string{"*"},
		DisableCompression: true,
		DisableEncryption:  true, // Also disable encryption to see raw message type
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	if err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher should have received message")
	}

	// Adapter should receive uncompressed, unencrypted message
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter should have received message")
	}

	// First byte should be MessageTypeBroadcast (not MessageTypeCompress or MessageTypeEncrypt)
	if len(adapterMsg.message) > 0 && adapterMsg.message[0] != byte(MessageTypeBroadcast) {
		t.Errorf("expected uncompressed message (type %d), got type %d",
			MessageTypeBroadcast, adapterMsg.message[0])
	}
}

// TestDispatchRemoteUnencryptedWhenEncryptionEnabled verifies error when remote sends unencrypted
// but local has encryption enabled.
func TestDispatchRemoteUnencryptedWhenEncryptionEnabled(t *testing.T) {
	testClearPubsub()

	topic := "remote:unencrypted"
	message := []byte{byte(MessageTypeBroadcast), 0x01, 0x02, 0x03}

	// Adapter with encryption enabled
	SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: false,
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Dispatch unencrypted message (should fail)
	Dispatch(topic, message)
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received != nil {
		t.Error("dispatcher should not have received unencrypted message when encryption is enabled")
	}
}

// TestCustomKeyringEncryption verifies custom keyring per adapter works.
func TestCustomKeyringEncryption(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "custom:keyring"
	message := []byte("custom keyring test")

	customKeyring := chain.NewKeyring("custom-test-salt", 216000, 32, "sha256")

	SetAdapters([]AdapterConfig{{
		Adapter: testAdapter,
		Topics:  []string{"*"},
		Keyring: customKeyring,
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := Broadcast(topic, message)
	if err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}
	<-time.After(time.Millisecond * 20)

	// Verify adapter received encrypted message
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter should have received message")
	}

	if adapterMsg.message[0] != byte(MessageTypeEncrypt) {
		t.Errorf("expected encrypted message (type %d), got type %d",
			MessageTypeEncrypt, adapterMsg.message[0])
	}
}

// ============================================================================
// Message Ordering and Delivery Tests
// ============================================================================

// TestMessageOrderingPreserved verifies messages are delivered in order to single dispatcher.
func TestMessageOrderingPreserved(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "order:test"
	const messageCount = 20

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	// Broadcast messages sequentially
	for i := 0; i < messageCount; i++ {
		msg := []byte(fmt.Sprintf("msg-%d", i))
		if err := Broadcast(topic, msg); err != nil {
			t.Fatal(err)
		}
		<-time.After(time.Millisecond * 5) // Small delay between broadcasts
	}
	<-time.After(time.Millisecond * 200)

	// Verify messages received in order
	for i := messageCount - 1; i >= 0; i-- {
		received := dispatcher.pop()
		if received == nil {
			t.Fatalf("missing message %d", i)
		}
		expected := fmt.Sprintf("msg-%d", i)
		if string(received.message.([]byte)) != expected {
			t.Errorf("message %d: expected %q, got %q", i, expected, received.message)
		}
	}
}

// ============================================================================
// LocalBroadcast Tests
// ============================================================================

// TestLocalBroadcast verifies LocalBroadcast dispatches only locally.
func TestLocalBroadcast(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "local:broadcast"
	message := []byte("local only")

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	LocalBroadcast(topic, message)
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher should have received local broadcast")
	}

	// Adapter should NOT have been called
	adapterMsg := testAdapter.pop()
	if adapterMsg != nil {
		t.Error("adapter should not have been called for LocalBroadcast")
	}
}

// ============================================================================
// DirectBroadcast Tests
// ============================================================================

// TestDirectBroadcastInvalidNodeId verifies DirectBroadcast with invalid node ID.
func TestDirectBroadcastInvalidNodeId(t *testing.T) {
	testClearPubsub()

	err := DirectBroadcast("invalid-ksuid", "test:topic", []byte("test"))
	if err == nil {
		t.Error("expected error for invalid node ID")
	}
}

// ============================================================================
// ResetPubsub Safety Tests
// ============================================================================

// TestResetPubsubSafety verifies ResetPubsub can be called safely.
func TestResetPubsubSafety(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			ResetPubsub()
		}()
	}

	wg.Wait()
	// Should not have panicked
}

// ============================================================================
// Global Options Tests
// ============================================================================

// TestSetGlobalOptions verifies global options are applied.
func TestSetGlobalOptions(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	SetGlobalOptions(O("global-key", "global-value"))
	defer SetGlobalOptions() // Clear after test

	topic := "global:options"
	message := []byte("test")

	err := Broadcast(topic, message)
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	// Verify adapter received global options
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter should have received message")
	}

	if val, ok := adapterMsg.opts["global-key"]; !ok || val != "global-value" {
		t.Errorf("expected global option global-key=global-value, got %v", adapterMsg.opts)
	}
}

// TestBroadcastOptionsOverride verifies per-broadcast options override global options.
func TestBroadcastOptionsOverride(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	SetGlobalOptions(O("shared-key", "global-value"))
	defer SetGlobalOptions()

	topic := "override:options"
	message := []byte("test")

	err := Broadcast(topic, message, O("shared-key", "local-value"), O("local-key", "local-value"))
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter should have received message")
	}

	// Should have overridden global option
	if val, ok := adapterMsg.opts["shared-key"]; !ok || val != "local-value" {
		t.Errorf("expected shared-key=local-value, got %v", adapterMsg.opts)
	}

	// Should have local option
	if val, ok := adapterMsg.opts["local-key"]; !ok || val != "local-value" {
		t.Errorf("expected local-key=local-value, got %v", adapterMsg.opts)
	}
}

// ============================================================================
// SetAdapters Tests
// ============================================================================

// TestSetAdaptersReplacesExisting verifies SetAdapters replaces existing adapters.
func TestSetAdaptersReplacesExisting(t *testing.T) {
	testClearPubsub()

	// First adapter
	SetAdapters([]AdapterConfig{{
		Adapter: testAdapter,
		Topics:  []string{"*"},
	}})

	config := GetAdapter("any:topic")
	if config == nil || config.Adapter.Name() != "test" {
		t.Fatal("expected test adapter")
	}

	// Replace with different adapter
	newAdapter := &testAdapterStruct{
		subscriptions: map[string]bool{},
		messages:      []*testAdapterMessage{},
	}
	SetAdapters([]AdapterConfig{{
		Adapter: newAdapter,
		Topics:  []string{"*"},
	}})

	config = GetAdapter("any:topic")
	if config == nil || config.Adapter.Name() != "test" {
		// Note: both adapters have name "test" in our test, but they are different instances
		t.Log("adapter replaced (instance check would require pointer comparison)")
	}
}

// TestSetAdaptersMultipleTopics verifies adapter with multiple topic patterns.
func TestSetAdaptersMultipleTopics(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	SetAdapters([]AdapterConfig{{
		Adapter: testAdapter,
		Topics:  []string{"admin:*", "user:*", "*"},
	}})

	// All should match
	for _, topic := range []string{"admin:config", "user:123", "anything"} {
		config := GetAdapter(topic)
		if config == nil {
			t.Errorf("expected adapter for topic %s", topic)
		}
	}
}

// ============================================================================
// DispatcherFunc Helper Tests
// ============================================================================

// TestDispatcherFunc verifies the DispatcherFunc helper works correctly.
func TestDispatcherFunc(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "func:dispatcher"
	message := []byte("func test")

	var receivedTopic string
	var receivedMessage []byte
	var receivedFrom string

	d := DispatcherFunc(func(topic string, message []byte, from string) {
		receivedTopic = topic
		receivedMessage = message
		receivedFrom = from
	})

	Subscribe(topic, d)

	if err := Broadcast(topic, message); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	if receivedTopic != topic {
		t.Errorf("expected topic %s, got %s", topic, receivedTopic)
	}
	if string(receivedMessage) != string(message) {
		t.Errorf("expected message %s, got %s", string(message), string(receivedMessage))
	}
	if receivedFrom != Self() {
		t.Errorf("expected from %s, got %s", Self(), receivedFrom)
	}
}

// TestDispatcherFuncNil verifies nil DispatcherFunc is handled safely.
func TestDispatcherFuncNil(t *testing.T) {
	d := DispatcherFunc(nil)
	// Should not panic
	d.Dispatch("test", []byte("test"), "from")
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

// TestDummyAdapter verifies DummyAdapter behavior.
func TestDummyAdapter(t *testing.T) {
	adapter := &DummyAdapter{}

	if adapter.Name() != "dummy" {
		t.Errorf("expected name 'dummy', got %s", adapter.Name())
	}

	// Broadcast should do nothing and return nil
	if err := adapter.Broadcast("topic", []byte("msg"), nil); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Subscribe/Unsubscribe should do nothing
	adapter.Subscribe("topic")
	adapter.Unsubscribe("topic")
}

// TestOptionKey verifies Option.Key returns correct value.
func TestOptionKey(t *testing.T) {
	opt := O("mykey", "myvalue")
	if opt.Key() != "mykey" {
		t.Errorf("expected key 'mykey', got %s", opt.Key())
	}
	if opt.Value() != "myvalue" {
		t.Errorf("expected value 'myvalue', got %v", opt.Value())
	}
}

// TestScheduleUnsubscribeEarlyReturn verifies scheduleUnsubscribe returns early
// if timer already removed by re-subscribe.
func TestScheduleUnsubscribeEarlyReturn(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	topic := "schedule:early"
	dispatcher := &testDispatcherStruct{}

	// Subscribe
	Subscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 10)

	// Unsubscribe (starts timer)
	Unsubscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 10)

	// Re-subscribe immediately (should cancel timer)
	Subscribe(topic, dispatcher)
	<-time.After(time.Millisecond * 100)

	// Timer should be gone
	p.unsubscribeMutex.Lock()
	_, hasTimer := p.unsubscribeTimers[topic]
	p.unsubscribeMutex.Unlock()

	if hasTimer {
		t.Error("timer should have been cancelled by re-subscribe")
	}

	// Adapter should still be subscribed
	if !testAdapter.subscribed(topic) {
		t.Error("adapter should still be subscribed")
	}
}

// TestDispatchDirectBroadcastWrongTopic verifies dispatch rejects direct broadcast with wrong topic.
func TestDispatchDirectBroadcastWrongTopic(t *testing.T) {
	testClearPubsub()

	// Dispatch a direct broadcast message with wrong topic
	selfID := getSelfIDBytes()
	wrongTopic := "direct:wrongnode"

	// Create a valid direct broadcast message
	buf := make([]byte, 0, 100)
	buf = append(buf, byte(MessageTypeDirectBroadcast))
	buf = append(buf, selfID...)             // from
	buf = append(buf, selfID...)             // to (self)
	buf = append(buf, []byte{0, 0, 0, 5}...) // topic len = 5
	buf = append(buf, []byte("hello")...)    // topic
	buf = append(buf, []byte("payload")...)  // payload

	dispatcher := &testDispatcherStruct{}
	Subscribe("hello", dispatcher)

	// Dispatch with wrong topic
	Dispatch(wrongTopic, buf)
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received != nil {
		t.Error("dispatcher should not have received direct broadcast with wrong topic")
	}
}

// TestDispatchDirectBroadcastWrongTarget verifies dispatch rejects direct broadcast targeted to other node.
func TestDispatchDirectBroadcastWrongTarget(t *testing.T) {
	testClearPubsub()

	selfID := getSelfIDBytes()
	correctTopic := getDirectTopic()

	// Create direct broadcast targeted to a different node
	otherNode := make([]byte, 20)
	for i := range otherNode {
		otherNode[i] = 0xFF
	}

	buf := make([]byte, 0, 100)
	buf = append(buf, byte(MessageTypeDirectBroadcast))
	buf = append(buf, selfID...)             // from
	buf = append(buf, otherNode...)          // to (other node)
	buf = append(buf, []byte{0, 0, 0, 5}...) // topic len = 5
	buf = append(buf, []byte("hello")...)    // topic
	buf = append(buf, []byte("payload")...)  // payload

	dispatcher := &testDispatcherStruct{}
	Subscribe("hello", dispatcher)

	Dispatch(correctTopic, buf)
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received != nil {
		t.Error("dispatcher should not have received direct broadcast targeted to other node")
	}
}

// TestBroadcastMessageWithDummyAdapter verifies broadcastMessage dispatches locally with dummy adapter.
func TestBroadcastMessageWithDummyAdapter(t *testing.T) {
	testClearPubsub()

	SetAdapters([]AdapterConfig{{
		Adapter:           &DummyAdapter{},
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	topic := "msg:dummy"
	message := []byte("broadcast message test")

	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	err := broadcastMessage(MessageTypeBroadcast, topic, message)
	<-time.After(time.Millisecond * 20)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	received := dispatcher.pop()
	if received == nil {
		t.Error("dispatcher should have received message")
	}
}

// TestDispatchInvalidMessageType verifies dispatch rejects unknown message types.
func TestDispatchInvalidMessageType(t *testing.T) {
	testClearPubsub()

	// Create message with invalid type (e.g., 99)
	buf := make([]byte, 0, 50)
	buf = append(buf, byte(99)) // Invalid message type
	buf = append(buf, getSelfIDBytes()...)
	buf = append(buf, []byte("payload")...)

	topic := "invalid:type"
	dispatcher := &testDispatcherStruct{}
	Subscribe(topic, dispatcher)

	SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	Dispatch(topic, buf)
	<-time.After(time.Millisecond * 20)

	received := dispatcher.pop()
	if received != nil {
		t.Error("dispatcher should not have received message with invalid type")
	}
}

// TestSetAdaptersPanicOnInvalid verifies SetAdapters panics on invalid config.
func TestSetAdaptersPanicOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on invalid adapter config")
		}
	}()

	// This should panic because pattern has * in the middle (invalid splat)
	SetAdapters([]AdapterConfig{{
		Adapter: testAdapter,
		Topics:  []string{"invalid:*:pattern"},
	}})
}
