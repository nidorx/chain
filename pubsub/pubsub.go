package pubsub

import (
	"sync"

	"github.com/segmentio/ksuid"
)

// ============================================================================
// Core Types
// ============================================================================

// Dispatcher interface must be implemented to receive subscribed messages.
type Dispatcher interface {
	Dispatch(topic string, message []byte, from string)
}

// DispatcherFuncImpl wraps a function to implement the Dispatcher interface.
type DispatcherFuncImpl struct {
	Dispatcher func(topic string, message []byte, from string)
}

func (d *DispatcherFuncImpl) Dispatch(topic string, message []byte, from string) {
	if d.Dispatcher == nil {
		return
	}
	d.Dispatcher(topic, message, from)
}

// DispatcherFunc wraps a function as a Dispatcher.
func DispatcherFunc(d func(topic string, message []byte, from string)) Dispatcher {
	return &DispatcherFuncImpl{Dispatcher: d}
}

// ============================================================================
// Legacy Global State and Backward Compatibility Layer
// ============================================================================

var (
	// selfId is the global node identifier (kept for backward compatibility)
	selfId       = ksuid.New()
	selfIdBytes  = selfId.Bytes()
	selfIdString = selfId.String()
	directTopic  = "direct:" + selfIdString
	selfIdMutex  sync.RWMutex

	// p is the legacy global pubsub instance.
	// It is initialized in init() to point to Default after Default is created.
	p *PubSub
)

func init() {
	// Make p point to Default so tests accessing p directly work correctly
	p = Default
}

// subscription represents the subscriptions that this server has.
type subscription struct {
	dispatchers map[Dispatcher]int // incremental dispatcher subscriptions
}

// ErrNoAdapter is returned when no adapter matches the broadcast topic.
var ErrNoAdapter = &pubsubError{"no adapter matches topic to broadcast the message"}

// MaxMessageSize limits message size to prevent memory exhaustion.
// Default: 1MB
var MaxMessageSize = 1 << 20

// pubsubError is a custom error type for pubsub operations.
type pubsubError struct {
	msg string
}

func (e *pubsubError) Error() string {
	return e.msg
}

// ============================================================================
// Test Helper Functions (exposed for testing)
// ============================================================================

// getSelfIDStringForTest returns selfIdString for testing purposes.
func getSelfIDStringForTest() string {
	selfIdMutex.RLock()
	defer selfIdMutex.RUnlock()
	return selfIdString
}

// getDirectTopicForTest returns directTopic for testing purposes.
func getDirectTopicForTest() string {
	selfIdMutex.RLock()
	defer selfIdMutex.RUnlock()
	return directTopic
}

// setSelfIDForTest updates the self ID variables for testing.
func setSelfIDForTest(id ksuid.KSUID) {
	selfIdMutex.Lock()
	defer selfIdMutex.Unlock()
	selfId = id
	selfIdBytes = id.Bytes()
	selfIdString = id.String()
	directTopic = "direct:" + selfIdString

	// Also update Default instance for consistency
	Default.selfIdMutex.Lock()
	Default.selfId = id
	Default.selfIdBytes = id.Bytes()
	Default.selfIdString = id.String()
	Default.selfIdMutex.Unlock()
}

// getSelfIDBytesForTest returns a copy of selfIdBytes for testing.
func getSelfIDBytesForTest() []byte {
	selfIdMutex.RLock()
	defer selfIdMutex.RUnlock()
	b := make([]byte, len(selfIdBytes))
	copy(b, selfIdBytes)
	return b
}

// broadcastMessageForTest is exposed for testing.
func broadcastMessageForTest(msgType MessageType, topic string, message []byte, options ...*Option) error {
	return Default.broadcastMessage(msgType, topic, message, options...)
}
