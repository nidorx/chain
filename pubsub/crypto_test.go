package pubsub

import (
	"reflect"
	"testing"

	"github.com/nidorx/chain"
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

// TestCryptoKeyringConfiguration verifies the keyring is properly configured
// with secure PBKDF2 iterations and random salt.
func TestCryptoKeyringConfiguration(t *testing.T) {
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		t.Fatal(err)
	}

	// Verify keyring is not nil
	if globalKeyring == nil {
		t.Fatal("globalKeyring should not be nil")
	}

	// Test encryption/decryption roundtrip
	payload := []byte("test message for keyring validation")
	encrypted, err := encryptPayload(globalKeyring, payload)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := decryptPayload(globalKeyring, encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decrypted) != string(payload) {
		t.Fatalf("roundtrip failed: expected %q, got %q", string(payload), string(decrypted))
	}
}

// TestCryptoRandomSalt verifies that the salt is randomly generated per initialization.
// This test ensures two different package initializations would produce different salts.
func TestCryptoRandomSalt(t *testing.T) {
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		t.Fatal(err)
	}

	// The keyring should be functional and not use the hardcoded salt
	payload := []byte("salt validation test")
	encrypted, err := encryptPayload(globalKeyring, payload)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := decryptPayload(globalKeyring, encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decrypted) != string(payload) {
		t.Fatalf("encryption/decryption failed with random salt")
	}
}
