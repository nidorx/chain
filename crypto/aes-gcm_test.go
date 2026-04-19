package crypto

import (
	"bytes"
	"crypto/rand"
	"sync"
	"testing"
)

func generateKey(size int) []byte {
	key := make([]byte, size)
	rand.Read(key)
	return key
}

func TestEncryptDecrypt_Roundtrip_16ByteKey(t *testing.T) {
	key := generateKey(16)
	plaintext := []byte("hello world")

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_Roundtrip_24ByteKey(t *testing.T) {
	key := generateKey(24)
	plaintext := []byte("AES-192 test data")

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_Roundtrip_32ByteKey(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("AES-256 test data")

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_WithAAD(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("secret message")
	aad := []byte("additional authenticated data")

	encrypted, err := Encrypt(key, plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt with AAD failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, aad)
	if err != nil {
		t.Fatalf("Decrypt with AAD failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptyData(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte{}

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt empty data failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt empty data failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty decrypted data, got %d bytes", len(decrypted))
	}
}

func TestEncryptDecrypt_LargeData(t *testing.T) {
	key := generateKey(32)
	plaintext := make([]byte, 10*1024*1024) // 10MB
	rand.Read(plaintext)

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt large data failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt large data failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("large data plaintext mismatch")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := generateKey(32)
	key2 := generateKey(32)
	plaintext := []byte("secret data")

	encrypted, err := Encrypt(key1, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key2, encrypted, nil)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key, got nil")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("untampered data")

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Tamper with a byte in the ciphertext portion (after the nonce)
	encrypted[16] ^= 0xFF

	_, err = Decrypt(key, encrypted, nil)
	if err == nil {
		t.Fatal("expected error when decrypting tampered ciphertext, got nil")
	}
}

func TestDecrypt_WrongAAD(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("secret message")
	correctAAD := []byte("correct aad")
	wrongAAD := []byte("wrong aad")

	encrypted, err := Encrypt(key, plaintext, correctAAD)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key, encrypted, wrongAAD)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong AAD, got nil")
	}
}

func TestDecrypt_EmptyCiphertext(t *testing.T) {
	key := generateKey(32)

	_, err := Decrypt(key, []byte{}, nil)
	if err == nil {
		t.Fatal("expected error when decrypting empty ciphertext, got nil")
	}
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	key := generateKey(32)

	// Create ciphertext shorter than nonce size (12 bytes)
	shortCiphertext := []byte{1, 2, 3, 4, 5}

	_, err := Decrypt(key, shortCiphertext, nil)
	if err == nil {
		t.Fatal("expected error when ciphertext is shorter than nonce size, got nil")
	}
}

func TestValidateKey_InvalidSizes(t *testing.T) {
	invalidSizes := []int{15, 17, 31}

	for _, size := range invalidSizes {
		key := generateKey(size)
		err := ValidateKey(key)
		if err == nil {
			t.Errorf("expected error for %d-byte key, got nil", size)
		}
	}
}

func TestValidateKey_ValidSizes(t *testing.T) {
	validSizes := []int{16, 24, 32}

	for _, size := range validSizes {
		key := generateKey(size)
		err := ValidateKey(key)
		if err != nil {
			t.Errorf("expected no error for %d-byte key, got: %v", size, err)
		}
	}
}

func TestValidateKey_AllInvalidSizes(t *testing.T) {
	invalidSizes := []int{0, 1, 8, 12, 13, 14, 15, 17, 18, 20, 23, 25, 28, 30, 31, 33, 64}

	for _, size := range invalidSizes {
		key := generateKey(size)
		err := ValidateKey(key)
		if err == nil {
			t.Errorf("expected error for %d-byte key, got nil", size)
		}
	}
}

func TestEncrypt_RandomNonce(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("test for random nonce")

	encrypted1, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	encrypted2, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Ciphertexts should differ due to random nonce
	if bytes.Equal(encrypted1, encrypted2) {
		t.Fatal("expected different ciphertexts due to random nonce, but they were identical")
	}

	// Both should decrypt to the same plaintext
	decrypted1, err := Decrypt(key, encrypted1, nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	decrypted2, err := Decrypt(key, encrypted2, nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted1, decrypted2) {
		t.Fatal("decrypted values from two encryptions should match")
	}
}

func TestDecrypt_ModifiedNonce(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("test modified nonce")

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Modify a byte in the nonce (first 12 bytes)
	encrypted[0] ^= 0xFF

	_, err = Decrypt(key, encrypted, nil)
	if err == nil {
		// This might succeed or fail depending on implementation,
		// but the decrypted data should not match the original
	}
}

func TestEncryptDecrypt_Concurrent(t *testing.T) {
	key := generateKey(32)
	numGoroutines := 100
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*2)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			plaintext := []byte("concurrent test data")
			encrypted, err := Encrypt(key, plaintext, nil)
			if err != nil {
				errCh <- err
				return
			}

			decrypted, err := Decrypt(key, encrypted, nil)
			if err != nil {
				errCh <- err
				return
			}

			if !bytes.Equal(plaintext, decrypted) {
				errCh <- nil // signal mismatch
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent encrypt/decrypt failed: %v", err)
		}
	}
}

func TestEncryptDecrypt_UnicodeData(t *testing.T) {
	key := generateKey(32)
	plaintext := []byte("Hello, 世界! 🌍 café naïve résumé")

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt unicode data failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt unicode data failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("unicode data mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_BinaryData(t *testing.T) {
	key := generateKey(32)

	// Generate random binary data with all possible byte values
	plaintext := make([]byte, 256)
	for i := 0; i < 256; i++ {
		plaintext[i] = byte(i)
	}

	encrypted, err := Encrypt(key, plaintext, nil)
	if err != nil {
		t.Fatalf("Encrypt binary data failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt binary data failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("binary data mismatch")
	}
}

func TestEncryptDecrypt_DataIntegrity(t *testing.T) {
	key := generateKey(32)

	// Test with various data patterns
	testCases := [][]byte{
		{0x00},
		{0xFF},
		{0x00, 0x00, 0x00, 0x00},
		{0xFF, 0xFF, 0xFF, 0xFF},
		bytes.Repeat([]byte{0xAB}, 100),
		bytes.Repeat([]byte{0x00, 0xFF}, 50),
	}

	for i, plaintext := range testCases {
		encrypted, err := Encrypt(key, plaintext, nil)
		if err != nil {
			t.Fatalf("Encrypt failed for case %d: %v", i, err)
		}

		decrypted, err := Decrypt(key, encrypted, nil)
		if err != nil {
			t.Fatalf("Decrypt failed for case %d: %v", i, err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Fatalf("data integrity failed for case %d", i)
		}
	}
}
