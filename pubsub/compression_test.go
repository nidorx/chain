package pubsub

import (
	"reflect"
	"testing"
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
