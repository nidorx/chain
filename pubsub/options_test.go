package pubsub

import (
	"testing"
	"time"
)

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

// TestTypedOptionsWithTypedOptions verifies typed options work with broadcast.
func TestTypedOptionsWithTypedOptions(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	// Use typed options in broadcast
	dispatcher := &testDispatcherStruct{}
	Subscribe("integration:typed", dispatcher)

	err := Broadcast("integration:typed", []byte("test"),
		WithCompression(true),
		WithTTL(60*time.Second),
	)

	// Should not error (test adapter accepts everything)
	if err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}
	<-time.After(time.Millisecond * 20)

	// Verify dispatcher received message
	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher did not receive message")
	}

	// Verify adapter received the options
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter should have received message")
	}

	// Check compression option
	if val, ok := adapterMsg.opts["compression"]; !ok || val != true {
		t.Errorf("expected compression=true, got %v", adapterMsg.opts)
	}

	// Check TTL option
	if _, ok := adapterMsg.opts["ttl"]; !ok {
		t.Errorf("expected ttl option to be present, got %v", adapterMsg.opts)
	}
}

// TestOptionAsBool verifies Option.AsBool conversion with defaults.
func TestOptionAsBool(t *testing.T) {
	tests := []struct {
		name         string
		option       *Option
		defaultValue bool
		expected     bool
	}{
		{"nil option", nil, true, true},
		{"nil option false", nil, false, false},
		{"bool true", O("test", true), false, true},
		{"bool false", O("test", false), true, false},
		{"string true", O("test", "true"), false, true},
		{"string false", O("test", "false"), true, false},
		{"int non-zero", O("test", 1), false, true},
		{"int zero", O("test", 0), true, false},
		{"int64 non-zero", O("test", int64(42)), false, true},
		{"int64 zero", O("test", int64(0)), true, false},
		{"float64 non-zero", O("test", 3.14), false, true},
		{"float64 zero", O("test", 0.0), true, false},
		{"unsupported type", O("test", []byte{1}), true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.option.AsBool(tt.defaultValue)
			if result != tt.expected {
				t.Errorf("AsBool(%v) = %v, want %v", tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestOptionAsInt verifies Option.AsInt conversion with defaults.
func TestOptionAsInt(t *testing.T) {
	tests := []struct {
		name         string
		option       *Option
		defaultValue int64
		expected     int64
	}{
		{"nil option", nil, 42, 42},
		{"int", O("test", 100), 0, 100},
		{"int64", O("test", int64(200)), 0, 200},
		{"uint", O("test", uint(300)), 0, 300},
		{"float64", O("test", 400.5), 0, 400},
		{"string", O("test", "500"), 0, 500},
		{"string invalid", O("test", "invalid"), 99, 99},
		{"bool true", O("test", true), 0, 1},
		{"bool false", O("test", false), 0, 0},
		{"unsupported type", O("test", []byte{1}), 77, 77},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.option.AsInt(tt.defaultValue)
			if result != tt.expected {
				t.Errorf("AsInt(%v) = %v, want %v", tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestOptionAsString verifies Option.AsString conversion with defaults.
func TestOptionAsString(t *testing.T) {
	tests := []struct {
		name         string
		option       *Option
		defaultValue string
		expected     string
	}{
		{"nil option", nil, "default", "default"},
		{"string value", O("test", "hello"), "default", "hello"},
		{"non-string value", O("test", 42), "default", "default"},
		{"empty string", O("test", ""), "default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.option.AsString(tt.defaultValue)
			if result != tt.expected {
				t.Errorf("AsString(%q) = %q, want %q", tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestOptionAsDuration verifies Option.AsDuration conversion with defaults.
func TestOptionAsDuration(t *testing.T) {
	defaultDur := 10 * time.Second

	tests := []struct {
		name         string
		option       *Option
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"nil option", nil, defaultDur, defaultDur},
		{"duration value", O("test", 5*time.Second), 0, 5 * time.Second},
		{"int64 nanoseconds", O("test", int64(1000000000)), 0, time.Second},
		{"int nanoseconds", O("test", 2000000000), 0, 2 * time.Second},
		{"float64 nanoseconds", O("test", 3000000000.0), 0, 3 * time.Second},
		{"string duration", O("test", "5s"), 0, 5 * time.Second},
		{"string invalid", O("test", "invalid"), defaultDur, defaultDur},
		{"unsupported type", O("test", []byte{1}), defaultDur, defaultDur},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.option.AsDuration(tt.defaultValue)
			if result != tt.expected {
				t.Errorf("AsDuration(%v) = %v, want %v", tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestWithCompression verifies WithCompression creates correct Option.
func TestWithCompression(t *testing.T) {
	opt := WithCompression(true)
	if opt.Key() != OptCompression {
		t.Errorf("expected key %q, got %q", OptCompression, opt.Key())
	}
	if !opt.AsBool(false) {
		t.Error("expected value true")
	}

	opt = WithCompression(false)
	if opt.AsBool(true) {
		t.Error("expected value false")
	}
}

// TestWithEncryption verifies WithEncryption creates correct Option.
func TestWithEncryption(t *testing.T) {
	opt := WithEncryption(true)
	if opt.Key() != OptEncryption {
		t.Errorf("expected key %q, got %q", OptEncryption, opt.Key())
	}
	if !opt.AsBool(false) {
		t.Error("expected value true")
	}
}

// TestWithTTL verifies WithTTL creates correct Option.
func TestWithTTL(t *testing.T) {
	ttl := 30 * time.Second
	opt := WithTTL(ttl)
	if opt.Key() != OptTTL {
		t.Errorf("expected key %q, got %q", OptTTL, opt.Key())
	}
	if opt.AsDuration(0) != ttl {
		t.Errorf("expected TTL %v, got %v", ttl, opt.AsDuration(0))
	}
}

// TestWithTimeout verifies WithTimeout creates correct Option.
func TestWithTimeout(t *testing.T) {
	timeout := 5 * time.Second
	opt := WithTimeout(timeout)
	if opt.Key() != OptTimeout {
		t.Errorf("expected key %q, got %q", OptTimeout, opt.Key())
	}
	if opt.AsDuration(0) != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, opt.AsDuration(0))
	}
}

// TestWithPriority verifies WithPriority creates correct Option.
func TestWithPriority(t *testing.T) {
	priority := "critical"
	opt := WithPriority(priority)
	if opt.Key() != OptPriority {
		t.Errorf("expected key %q, got %q", OptPriority, opt.Key())
	}
	if opt.AsString("normal") != priority {
		t.Errorf("expected priority %q, got %q", priority, opt.AsString("normal"))
	}
}

// TestOptionConstants verifies all option constants are defined.
func TestOptionConstants(t *testing.T) {
	if OptCompression == "" {
		t.Error("OptCompression should not be empty")
	}
	if OptEncryption == "" {
		t.Error("OptEncryption should not be empty")
	}
	if OptTTL == "" {
		t.Error("OptTTL should not be empty")
	}
	if OptTimeout == "" {
		t.Error("OptTimeout should not be empty")
	}
	if OptPriority == "" {
		t.Error("OptPriority should not be empty")
	}
}

// TestTypedOptionsIntegration verifies typed options work with broadcast.
func TestTypedOptionsIntegration(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	// Use typed options in broadcast
	dispatcher := &testDispatcherStruct{}
	Subscribe("integration:typed", dispatcher)

	err := Broadcast("integration:typed", []byte("test"),
		WithCompression(true),
		WithEncryption(true),
		WithTTL(60*time.Second),
	)

	// Should not error (test adapter accepts everything)
	if err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}
	<-time.After(time.Millisecond * 20)

	// Verify dispatcher received message
	received := dispatcher.pop()
	if received == nil {
		t.Fatal("dispatcher did not receive message")
	}
}

// TestOptionMultipleTypes verifies options can hold different types.
func TestOptionMultipleTypes(t *testing.T) {
	// Bool option
	boolOpt := O("bool", true)
	if !boolOpt.AsBool(false) {
		t.Error("bool option should return true")
	}

	// Int option
	intOpt := O("int", 42)
	if intOpt.AsInt(0) != 42 {
		t.Errorf("int option should return 42, got %d", intOpt.AsInt(0))
	}

	// String option
	strOpt := O("string", "hello")
	if strOpt.AsString("default") != "hello" {
		t.Errorf("string option should return 'hello', got %s", strOpt.AsString("default"))
	}

	// Duration option
	durOpt := O("duration", 5*time.Second)
	if durOpt.AsDuration(0) != 5*time.Second {
		t.Errorf("duration option should return 5s, got %v", durOpt.AsDuration(0))
	}
}

// TestOptionAsBoolWithInvalidString verifies AsBool handles invalid strings.
func TestOptionAsBoolWithInvalidString(t *testing.T) {
	opt := O("test", "invalid")
	result := opt.AsBool(true)
	if !result {
		t.Error("AsBool should return default value for invalid string")
	}
}

// TestOptionAsIntWithInvalidString verifies AsInt handles invalid strings.
func TestOptionAsIntWithInvalidString(t *testing.T) {
	opt := O("test", "not-a-number")
	result := opt.AsInt(99)
	if result != 99 {
		t.Errorf("AsInt should return default value for invalid string, got %d", result)
	}
}

// TestOptionAsDurationWithInvalidString verifies AsDuration handles invalid strings.
func TestOptionAsDurationWithInvalidString(t *testing.T) {
	defaultDur := 5 * time.Second
	opt := O("test", "not-a-duration")
	result := opt.AsDuration(defaultDur)
	if result != defaultDur {
		t.Errorf("AsDuration should return default value for invalid string, got %v", result)
	}
}
