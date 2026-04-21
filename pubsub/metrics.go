package pubsub

import "time"

// MetricsCollector defines the interface for collecting pubsub observability data.
//
// Implementations can integrate with Prometheus, OpenTelemetry, or custom
// monitoring systems. A no-op implementation is used by default.
//
// Example: Prometheus Integration
//
//	type PrometheusMetrics struct {
//	    messagesSent     *prometheus.CounterVec
//	    messagesReceived *prometheus.CounterVec
//	    dispatchDuration *prometheus.HistogramVec
//	    errorCount       *prometheus.CounterVec
//	}
//
//	func (m *PrometheusMetrics) MessageSent(topic string, size int, duration time.Duration) {
//	    m.messagesSent.WithLabelValues(topic).Inc()
//	}
//
// Usage:
//
//	metrics := NewPrometheusMetrics()
//	ps.SetMetricsCollector(metrics)
type MetricsCollector interface {
	// MessageSent is called when a message is successfully sent to an adapter.
	//
	// Parameters:
	//   - topic: The topic the message was sent to
	//   - size: The size of the message in bytes (after compression/encryption)
	//   - duration: Time taken to send the message (0 if not measured)
	MessageSent(topic string, size int, duration time.Duration)

	// MessageReceived is called when a message is received from an adapter.
	//
	// Parameters:
	//   - topic: The topic the message was received on
	//   - size: The size of the message in bytes (before decompression/decryption)
	//   - from: The node ID of the sender
	MessageReceived(topic string, size int, from string)

	// Dispatched is called when a message is dispatched to local subscribers.
	//
	// Parameters:
	//   - topic: The topic being dispatched
	//   - subscriberCount: Number of subscribers that received the message
	//   - duration: Time taken to dispatch (0 if not measured)
	Dispatched(topic string, subscriberCount int, duration time.Duration)

	// SubscribeCount is called when the subscription count for a topic changes.
	//
	// Parameters:
	//   - topic: The topic pattern
	//   - count: Current number of dispatchers subscribed to this topic
	SubscribeCount(topic string, count int)

	// Error is called on any error condition.
	//
	// Parameters:
	//   - operation: The operation that failed (e.g., "broadcast", "dispatch", "encrypt")
	//   - topic: The topic associated with the error (may be empty)
	//   - err: The error that occurred
	Error(operation string, topic string, err error)
}

// noopMetrics is a no-op implementation of MetricsCollector.
// Used as the default when no metrics collector is configured.
type noopMetrics struct{}

func (n *noopMetrics) MessageSent(string, int, time.Duration) {}
func (n *noopMetrics) MessageReceived(string, int, string)    {}
func (n *noopMetrics) Dispatched(string, int, time.Duration)  {}
func (n *noopMetrics) SubscribeCount(string, int)             {}
func (n *noopMetrics) Error(string, string, error)            {}

// ============================================================================
// Global metrics collector (for backward compatibility)
// ============================================================================

// globalMetrics holds the global metrics collector instance.
// New instances should use SetMetricsCollector() method instead.
var globalMetrics MetricsCollector = &noopMetrics{}

// SetMetricsCollector configures the global metrics collector.
//
// This affects the Default instance. For instance-specific metrics collection,
// use ps.SetMetricsCollector() instead.
func SetMetricsCollector(m MetricsCollector) {
	if m == nil {
		globalMetrics = &noopMetrics{}
	} else {
		globalMetrics = m
		Default.SetMetricsCollector(m)
	}
}
