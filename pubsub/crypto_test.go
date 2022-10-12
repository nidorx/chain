package pubsub

import (
	"github.com/syntax-framework/chain"
	"reflect"
	"testing"
)

func Test_PubSub_Crypto(t *testing.T) {
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}

	for _, tt := range testPayloads {
		t.Run(tt.content, func(t *testing.T) {
			payload := []byte(tt.content)
			encrypted, err := encryptPayload(globalKeyring, payload)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}

			//fmt.Printf("Len from %d to %d", len(payload), len(compressed))

			decrypted, err := decryptPayload(globalKeyring, encrypted)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}

			if !reflect.DeepEqual(decrypted, payload) {
				t.Fatalf("bad payload: %v", decrypted)
			}
		})
	}
}
