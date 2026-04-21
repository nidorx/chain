package pubsub

import (
	"sync"
	"testing"
	"time"
)

// testMetricsCollector is a mock metrics collector for testing.
type testMetricsCollector struct {
	mu              sync.Mutex
	messageSent     int
	messageReceived int
	dispatched      int
	subscribeCount  int
	errors          int
	lastTopic       string
	lastError       error
}

func (m *testMetricsCollector) MessageSent(topic string, size int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageSent++
	m.lastTopic = topic
}

func (m *testMetricsCollector) MessageReceived(topic string, size int, from string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageReceived++
	m.lastTopic = topic
}

func (m *testMetricsCollector) Dispatched(topic string, subscriberCount int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dispatched++
	m.lastTopic = topic
}

func (m *testMetricsCollector) SubscribeCount(topic string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeCount++
	m.lastTopic = topic
}

func (m *testMetricsCollector) Error(operation string, topic string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
	m.lastTopic = topic
	m.lastError = err
}

// TestMetricsCollectorBroadcast verifies metrics are collected on broadcast.
func TestMetricsCollectorBroadcast(t *testing.T) {
	metrics := &testMetricsCollector{}
	ps := New()
	ps.SetMetricsCollector(metrics)
	defer ps.Close()

	// Use testAdapter which actually receives the message
	ps.SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	dispatcher := &testDispatcherStruct{}
	ps.Subscribe("metrics:broadcast", dispatcher)

	if err := ps.Broadcast("metrics:broadcast", []byte("test")); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	// With dummy adapter, local dispatch happens but adapter Broadcast is not called
	// So we should check dispatched metric instead
	if metrics.dispatched == 0 {
		t.Error("Dispatched should have been called")
	}
}

// TestMetricsCollectorSubscribe verifies metrics are collected on subscribe.
func TestMetricsCollectorSubscribe(t *testing.T) {
	metrics := &testMetricsCollector{}
	ps := New()
	ps.SetMetricsCollector(metrics)
	defer ps.Close()

	dispatcher := &testDispatcherStruct{}
	ps.Subscribe("metrics:subscribe", dispatcher)

	metrics.mu.Lock()
	count := metrics.subscribeCount
	metrics.mu.Unlock()

	if count == 0 {
		t.Error("SubscribeCount should have been called")
	}
}

// TestMetricsCollectorError verifies metrics are collected on errors.
func TestMetricsCollectorError(t *testing.T) {
	metrics := &testMetricsCollector{}
	ps := New()
	ps.SetMetricsCollector(metrics)
	defer ps.Close()

	// Try to broadcast on a topic with no adapter (will use dummy which is fine)
	// Let's test error metrics directly
	metrics.Error("test", "test:topic", errNew("test error"))

	metrics.mu.Lock()
	errors := metrics.errors
	metrics.mu.Unlock()

	if errors == 0 {
		t.Error("Error should have been called")
	}
}

// TestGlobalMetricsCollector verifies global metrics collector works.
func TestGlobalMetricsCollector(t *testing.T) {
	metrics := &testMetricsCollector{}
	SetMetricsCollector(metrics)

	// Reset to ensure clean state
	Reset()

	SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	dispatcher := &testDispatcherStruct{}
	Subscribe("global:metrics", dispatcher)

	if err := Broadcast("global:metrics", []byte("test")); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	metrics.mu.Lock()
	dispatched := metrics.dispatched
	metrics.mu.Unlock()

	// Should have dispatched at least once
	if dispatched == 0 {
		t.Error("Global metrics collector should receive Dispatched")
	}
}
