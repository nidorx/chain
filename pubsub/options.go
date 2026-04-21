package pubsub

import (
	"strconv"
	"time"
)

var (
	globalOptions = map[string]any{}
)

// Option key constants for typed broadcast options.
// These constants provide type-safe option keys that work with the existing Option system.
const (
	// OptCompression enables/disables message compression.
	// Value type: bool
	// Default: uses adapter's DisableCompression setting
	OptCompression = "compression"

	// OptEncryption enables/disables message encryption.
	// Value type: bool
	// Default: uses adapter's DisableEncryption setting
	OptEncryption = "encryption"

	// OptTTL sets the message time-to-live.
	// Value type: time.Duration (stored as int64 nanoseconds)
	// Default: no TTL (message never expires)
	OptTTL = "ttl"

	// OptTimeout sets the maximum time for broadcast operation.
	// Value type: time.Duration (stored as int64 nanoseconds)
	// Default: no timeout
	OptTimeout = "timeout"

	// OptPriority sets the message priority level.
	// Value type: string
	// Common values: "low", "normal", "high", "critical"
	// Default: "normal"
	OptPriority = "priority"
)

type Option struct {
	key   string
	value any
}

func (o *Option) Key() string {
	return o.key
}

func (o *Option) Value() any {
	return o.value
}

func O(key string, value any) *Option {
	return &Option{key, value}
}

// SetGlobalOptions set global options for sending messages
func SetGlobalOptions(options ...*Option) {
	for _, option := range options {
		key := option.key
		value := option.value
		globalOptions[key] = value
	}
}

// AsBool returns the option value as a bool.
// Returns the default value if the option is nil or conversion fails.
func (o *Option) AsBool(defaultValue bool) bool {
	if o == nil {
		return defaultValue
	}
	switch v := o.value.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return defaultValue
		}
		return b
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return defaultValue
	}
}

// AsInt returns the option value as an int64.
// Returns the default value if the option is nil or conversion fails.
func (o *Option) AsInt(defaultValue int64) int64 {
	if o == nil {
		return defaultValue
	}
	switch v := o.value.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return defaultValue
		}
		return i
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		return defaultValue
	}
}

// AsString returns the option value as a string.
// Returns the default value if the option is nil or conversion fails.
func (o *Option) AsString(defaultValue string) string {
	if o == nil {
		return defaultValue
	}
	if s, ok := o.value.(string); ok {
		return s
	}
	return defaultValue
}

// AsDuration returns the option value as a time.Duration.
// Returns the default value if the option is nil or conversion fails.
func (o *Option) AsDuration(defaultValue time.Duration) time.Duration {
	if o == nil {
		return defaultValue
	}
	switch v := o.value.(type) {
	case time.Duration:
		return v
	case int64:
		return time.Duration(v)
	case int:
		return time.Duration(v)
	case float64:
		return time.Duration(v)
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return defaultValue
		}
		return d
	default:
		return defaultValue
	}
}

// WithCompression creates an Option to enable/disable message compression.
//
// Example:
//
//	pubsub.Broadcast("user:123", msg, pubsub.WithCompression(true))
func WithCompression(enabled bool) *Option {
	return O(OptCompression, enabled)
}

// WithEncryption creates an Option to enable/disable message encryption.
//
// Example:
//
//	pubsub.Broadcast("user:123", msg, pubsub.WithEncryption(false))
func WithEncryption(enabled bool) *Option {
	return O(OptEncryption, enabled)
}

// WithTTL creates an Option to set the message time-to-live.
//
// Adapters may use this to automatically expire messages after the TTL.
// This is useful for transient messages that shouldn't be replayed.
//
// Example:
//
//	pubsub.Broadcast("user:123", msg, pubsub.WithTTL(30*time.Second))
func WithTTL(ttl time.Duration) *Option {
	return O(OptTTL, ttl)
}

// WithTimeout creates an Option to set the maximum time for broadcast operation.
//
// Adapters may use this to enforce operation timeouts.
//
// Example:
//
//	pubsub.Broadcast("user:123", msg, pubsub.WithTimeout(5*time.Second))
func WithTimeout(timeout time.Duration) *Option {
	return O(OptTimeout, timeout)
}

// WithPriority creates an Option to set the message priority level.
//
// Adapters may use this for message ordering or queue prioritization.
// Common values: "low", "normal", "high", "critical"
//
// Example:
//
//	pubsub.Broadcast("alerts:critical", msg, pubsub.WithPriority("critical"))
func WithPriority(priority string) *Option {
	return O(OptPriority, priority)
}
