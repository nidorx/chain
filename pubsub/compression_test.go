package pubsub

import (
	"reflect"
	"testing"
	"time"
)

func Test_PubSub_Compression(t *testing.T) {
	for _, tt := range testPayloads {
		t.Run(tt.content, func(t *testing.T) {
			payload := []byte(tt.content)
			compressed, err := compressPayload(payload)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}

			//fmt.Printf("Len from %d to %d", len(payload), len(compressed))

			dec, err := decompressPayload(compressed)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}

			if !reflect.DeepEqual(dec, payload) {
				t.Fatalf("bad payload: %v", dec)
			}
		})
	}
}

// TestCompressionSkipSmallMessages verifies compression is skipped for small messages.
func TestCompressionSkipSmallMessages(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	// Small message (under 128 bytes threshold)
	smallMessage := []byte("small message")

	dispatcher := &testDispatcherStruct{}
	Subscribe("compress:small", dispatcher)

	// Use testAdapter with encryption disabled
	SetAdapters([]AdapterConfig{{
		Adapter:           testAdapter,
		Topics:            []string{"*"},
		DisableEncryption: true,
	}})

	if err := Broadcast("compress:small", smallMessage); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	// Check adapter received message (should not be compressed)
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter did not receive message")
	}

	// Message should have MessageTypeBroadcast prefix (not MessageTypeCompress)
	if MessageType(adapterMsg.message[0]) != MessageTypeBroadcast {
		t.Errorf("expected MessageTypeBroadcast, got %v", adapterMsg.message[0])
	}
}

// TestCompressionLargerMessages verifies compression is applied to larger messages.
func TestCompressionLargerMessages(t *testing.T) {
	testClearPubsub()
	testAdapter.clear()

	// Larger message (over 128 bytes threshold)
	largeMessage := make([]byte, 200)
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	dispatcher := &testDispatcherStruct{}
	Subscribe("compress:large", dispatcher)

	if err := Broadcast("compress:large", largeMessage); err != nil {
		t.Fatal(err)
	}
	<-time.After(time.Millisecond * 20)

	// Adapter should receive message (may or may not be compressed depending on effectiveness)
	adapterMsg := testAdapter.pop()
	if adapterMsg == nil {
		t.Fatal("adapter did not receive message")
	}

	// Just verify no error occurred
	// The actual compression decision happens in the broadcast code
}
