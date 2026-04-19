package crypto

import (
	"bytes"
	"sync"
	"testing"
)

// 1. AddKey with valid 16-byte key
func TestKeyring_AddKey_16Bytes(t *testing.T) {
	k := &Keyring{}
	key := generateKey(16)

	err := k.AddKey(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	keys := k.GetKeys()
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if !bytes.Equal(keys[0], key) {
		t.Fatal("stored key does not match added key")
	}
}

// 2. AddKey with valid 24-byte key
func TestKeyring_AddKey_24Bytes(t *testing.T) {
	k := &Keyring{}
	key := generateKey(24)

	err := k.AddKey(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	keys := k.GetKeys()
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if !bytes.Equal(keys[0], key) {
		t.Fatal("stored key does not match added key")
	}
}

// 3. AddKey with valid 32-byte key
func TestKeyring_AddKey_32Bytes(t *testing.T) {
	k := &Keyring{}
	key := generateKey(32)

	err := k.AddKey(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	keys := k.GetKeys()
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if !bytes.Equal(keys[0], key) {
		t.Fatal("stored key does not match added key")
	}
}

// 4. AddKey with invalid key size (returns error)
func TestKeyring_AddKey_InvalidKeySize(t *testing.T) {
	k := &Keyring{}

	invalidSizes := []int{0, 1, 8, 15, 20, 31, 33, 64}
	for _, size := range invalidSizes {
		key := generateKey(size)
		err := k.AddKey(key)
		if err == nil {
			t.Errorf("expected error for key size %d, got nil", size)
		}
		if err != ErrKeySize {
			t.Errorf("expected ErrKeySize for key size %d, got %v", size, err)
		}
	}
}

// 5. AddKey duplicate (no-op)
func TestKeyring_AddKey_Duplicate_NoOp(t *testing.T) {
	k := &Keyring{}
	key := generateKey(32)

	err := k.AddKey(key)
	if err != nil {
		t.Fatalf("first AddKey failed: %v", err)
	}

	err = k.AddKey(key)
	if err != nil {
		t.Fatalf("duplicate AddKey should be no-op, got error: %v", err)
	}

	keys := k.GetKeys()
	if len(keys) != 1 {
		t.Fatalf("expected 1 key after duplicate add, got %d", len(keys))
	}
}

// 6. AddKey nil/empty keyring initialization
func TestKeyring_AddKey_EmptyKeyring_Init(t *testing.T) {
	k := &Keyring{}
	key := generateKey(32)

	// Verify keyring starts empty
	if len(k.GetKeys()) != 0 {
		t.Fatal("new keyring should have no keys")
	}

	err := k.AddKey(key)
	if err != nil {
		t.Fatalf("AddKey to empty keyring failed: %v", err)
	}

	if len(k.GetKeys()) != 1 {
		t.Fatal("keyring should have one key after AddKey")
	}
}

// 7. GetPrimaryKey returns first added key
func TestKeyring_GetPrimaryKey_ReturnsFirstAdded(t *testing.T) {
	k := &Keyring{}
	key1 := generateKey(16)
	key2 := generateKey(24)
	key3 := generateKey(32)

	k.AddKey(key1)
	primary := k.GetPrimaryKey()
	if !bytes.Equal(primary, key1) {
		t.Fatal("primary key should be key1")
	}

	// Adding more keys makes the new one primary (prepended)
	k.AddKey(key2)
	primary = k.GetPrimaryKey()
	if !bytes.Equal(primary, key2) {
		t.Fatal("primary key should be key2 after adding")
	}

	k.AddKey(key3)
	primary = k.GetPrimaryKey()
	if !bytes.Equal(primary, key3) {
		t.Fatal("primary key should be key3 after adding")
	}
}

// 8. GetKeys returns all keys in order
func TestKeyring_GetKeys_Order(t *testing.T) {
	k := &Keyring{}
	key1 := generateKey(16)
	key2 := generateKey(24)
	key3 := generateKey(32)

	k.AddKey(key1)
	k.AddKey(key2)
	k.AddKey(key3)

	keys := k.GetKeys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	// AddKey prepends, so order is [key3, key2, key1]
	if !bytes.Equal(keys[0], key3) {
		t.Fatal("keys[0] should be key3 (most recent)")
	}
	if !bytes.Equal(keys[1], key2) {
		t.Fatal("keys[1] should be key2")
	}
	if !bytes.Equal(keys[2], key1) {
		t.Fatal("keys[2] should be key1 (first added)")
	}
}

// 9. Encrypt/Decrypt roundtrip with single key
func TestKeyring_EncryptDecrypt_SingleKey(t *testing.T) {
	k := &Keyring{}
	k.AddKey(generateKey(32))

	data := []byte("hello world")
	aad := []byte("associated data")

	encrypted, err := k.Encrypt(data, aad)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := k.Decrypt(encrypted, aad)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, data) {
		t.Fatalf("decrypted data does not match original: got %q, want %q", decrypted, data)
	}
}

// 10. Encrypt with primary, decrypt after rotation (multiple keys)
func TestKeyring_EncryptDecrypt_AfterRotation(t *testing.T) {
	k := &Keyring{}
	key1 := generateKey(32)
	k.AddKey(key1)

	data := []byte("sensitive data")
	aad := []byte("aad")

	encrypted, err := k.Encrypt(data, aad)
	if err != nil {
		t.Fatalf("Encrypt with key1 failed: %v", err)
	}

	// Add new key (becomes primary)
	key2 := generateKey(32)
	k.AddKey(key2)

	// Decrypt should still work with old data
	decrypted, err := k.Decrypt(encrypted, aad)
	if err != nil {
		t.Fatalf("Decrypt after rotation failed: %v", err)
	}

	if !bytes.Equal(decrypted, data) {
		t.Fatal("decrypted data does not match original after key rotation")
	}
}

// 11. Key rotation: add new key, old data still decryptable
func TestKeyring_KeyRotation_OldDataDecryptable(t *testing.T) {
	k := &Keyring{}
	k.AddKey(generateKey(32))

	originalData := []byte("data before rotation")
	aad := []byte("rotation test aad")

	cipherText, err := k.Encrypt(originalData, aad)
	if err != nil {
		t.Fatalf("Encrypt before rotation failed: %v", err)
	}

	// Rotate key
	k.AddKey(generateKey(32))

	// Old ciphertext should still decrypt
	plainText, err := k.Decrypt(cipherText, aad)
	if err != nil {
		t.Fatalf("Decrypt old data after rotation failed: %v", err)
	}

	if !bytes.Equal(plainText, originalData) {
		t.Fatal("old data decryption produced different plaintext")
	}
}

// 12. Remove old key scenario (data encrypted with removed key fails)
func TestKeyring_RemoveKey_DataFails(t *testing.T) {
	k := &Keyring{}
	key1 := generateKey(32)
	key2 := generateKey(32)
	key3 := generateKey(32)

	// Build keyring: key3, key2, key1 (key3 is primary)
	k.AddKey(key1)
	k.AddKey(key2)
	k.AddKey(key3)

	data := []byte("data encrypted with key1")
	aad := []byte("remove test")

	// Create a separate keyring with only key1 to encrypt
	k1 := &Keyring{}
	k1.AddKey(key1)

	encrypted, err := k1.Encrypt(data, aad)
	if err != nil {
		t.Fatalf("Encrypt with key1 failed: %v", err)
	}

	// Verify it decrypts with all keys present
	_, err = k.Decrypt(encrypted, aad)
	if err != nil {
		t.Fatalf("Decrypt with all keys present failed: %v", err)
	}

	// Create a new keyring without key1
	kWithoutKey1 := &Keyring{}
	kWithoutKey1.AddKey(key3)
	kWithoutKey1.AddKey(key2)

	// Decryption should fail
	_, err = kWithoutKey1.Decrypt(encrypted, aad)
	if err == nil {
		t.Fatal("expected error when decrypting with removed key, got nil")
	}
	if err != ErrKeyringCannotDecrypt {
		t.Fatalf("expected ErrKeyringCannotDecrypt, got %v", err)
	}
}

// 13. Decrypt with wrong key (all keys fail)
func TestKeyring_Decrypt_WrongKey_AllFail(t *testing.T) {
	k := &Keyring{}
	k.AddKey(generateKey(32))

	// Encrypt with a different keyring
	otherK := &Keyring{}
	otherK.AddKey(generateKey(32))

	data := []byte("secret")
	aad := []byte("aad")

	encrypted, err := otherK.Encrypt(data, aad)
	if err != nil {
		t.Fatalf("Encrypt with other key failed: %v", err)
	}

	// Decrypt should fail
	_, err = k.Decrypt(encrypted, aad)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key, got nil")
	}
	if err != ErrKeyringCannotDecrypt {
		t.Fatalf("expected ErrKeyringCannotDecrypt, got %v", err)
	}
}

// 14. MessageEncrypt/MessageDecrypt roundtrip
func TestKeyring_MessageEncryptDecrypt_Roundtrip(t *testing.T) {
	k := &Keyring{}
	k.AddKey(generateKey(32))

	content := []byte("message content")
	aad := []byte("message aad")

	encrypted, err := k.MessageEncrypt(content, aad)
	if err != nil {
		t.Fatalf("MessageEncrypt failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("encrypted message should not be empty")
	}

	decrypted, err := k.MessageDecrypt([]byte(encrypted), aad)
	if err != nil {
		t.Fatalf("MessageDecrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Fatalf("decrypted content does not match: got %q, want %q", decrypted, content)
	}
}

// 15. MessageEncrypt with empty keyring (error)
func TestKeyring_MessageEncrypt_EmptyKeyring(t *testing.T) {
	k := &Keyring{}

	_, err := k.MessageEncrypt([]byte("data"), []byte("aad"))
	if err == nil {
		t.Fatal("expected error with empty keyring, got nil")
	}
	if err != ErrKeyringEmpty {
		t.Fatalf("expected ErrKeyringEmpty, got %v", err)
	}
}

// 16. MessageSign/MessageVerify roundtrip
func TestKeyring_MessageSignVerify_Roundtrip(t *testing.T) {
	k := &Keyring{}
	k.AddKey(generateKey(32))

	message := []byte("sign this message")
	digest := "sha256"

	signed, err := k.MessageSign(message, digest)
	if err != nil {
		t.Fatalf("MessageSign failed: %v", err)
	}

	if signed == "" {
		t.Fatal("signed message should not be empty")
	}

	verified, err := k.MessageVerify([]byte(signed))
	if err != nil {
		t.Fatalf("MessageVerify failed: %v", err)
	}

	if !bytes.Equal(verified, message) {
		t.Fatalf("verified message does not match: got %q, want %q", verified, message)
	}
}

// 17. MessageSign with empty keyring (error)
func TestKeyring_MessageSign_EmptyKeyring(t *testing.T) {
	k := &Keyring{}

	_, err := k.MessageSign([]byte("message"), "sha256")
	if err == nil {
		t.Fatal("expected error with empty keyring, got nil")
	}
	if err != ErrKeyringEmpty {
		t.Fatalf("expected ErrKeyringEmpty, got %v", err)
	}
}

// 18. MessageVerify with rotated keys (succeeds with any key)
func TestKeyring_MessageVerify_RotatedKeys(t *testing.T) {
	k := &Keyring{}
	key1 := generateKey(32)
	k.AddKey(key1)

	message := []byte("message for rotation test")
	digest := "sha256"

	// Sign with key1
	signed, err := k.MessageSign(message, digest)
	if err != nil {
		t.Fatalf("MessageSign failed: %v", err)
	}

	// Rotate key
	k.AddKey(generateKey(32))

	// Verify should still succeed with old signature
	verified, err := k.MessageVerify([]byte(signed))
	if err != nil {
		t.Fatalf("MessageVerify after rotation failed: %v", err)
	}

	if !bytes.Equal(verified, message) {
		t.Fatal("verified message does not match after rotation")
	}
}

// 19. Concurrent key operations (thread safety)
func TestKeyring_Concurrent_Operations(t *testing.T) {
	k := &Keyring{}
	var wg sync.WaitGroup

	// Concurrently add keys
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := generateKey(32)
			_ = k.AddKey(key)
		}(i)
	}

	// Concurrently read keys
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = k.GetKeys()
			_ = k.GetPrimaryKey()
		}()
	}

	// Concurrent encrypt/decrypt
	k.AddKey(generateKey(32))
	data := []byte("concurrent test data")
	aad := []byte("concurrent aad")

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			encrypted, err := k.Encrypt(data, aad)
			if err != nil {
				t.Errorf("concurrent encrypt failed: %v", err)
				return
			}
			decrypted, err := k.Decrypt(encrypted, aad)
			if err != nil {
				t.Errorf("concurrent decrypt failed: %v", err)
				return
			}
			if !bytes.Equal(decrypted, data) {
				t.Error("concurrent decrypt mismatch")
			}
		}()
	}

	wg.Wait()
}

// 20. GetPrimaryKey from empty keyring (returns nil)
func TestKeyring_GetPrimaryKey_Empty(t *testing.T) {
	k := &Keyring{}

	primary := k.GetPrimaryKey()
	if primary != nil {
		t.Fatalf("expected nil primary key from empty keyring, got %v", primary)
	}
}

// 21. Multiple key rotation scenario (3+ keys)
func TestKeyring_MultipleKeyRotation(t *testing.T) {
	k := &Keyring{}

	keys := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		keys[i] = generateKey(32)
		k.AddKey(keys[i])
	}

	allKeys := k.GetKeys()
	if len(allKeys) != 5 {
		t.Fatalf("expected 5 keys, got %d", len(allKeys))
	}

	// Verify order: most recently added is first
	for i := 0; i < 5; i++ {
		if !bytes.Equal(allKeys[i], keys[4-i]) {
			t.Fatalf("key at index %d mismatch", i)
		}
	}

	// Primary should be the last added key
	primary := k.GetPrimaryKey()
	if !bytes.Equal(primary, keys[4]) {
		t.Fatal("primary key should be the last added key")
	}
}

// 22. Encrypt after key rotation uses new primary
func TestKeyring_EncryptAfterRotation_UsesNewPrimary(t *testing.T) {
	k := &Keyring{}
	key1 := generateKey(32)
	key2 := generateKey(32)

	k.AddKey(key1)

	// Encrypt with key1
	data1 := []byte("data1")
	encrypted1, _ := k.Encrypt(data1, nil)

	// Rotate to key2
	k.AddKey(key2)

	// Encrypt again - should use key2
	data2 := []byte("data2")
	encrypted2, err := k.Encrypt(data2, nil)
	if err != nil {
		t.Fatalf("Encrypt after rotation failed: %v", err)
	}

	// Create a keyring with only key2 to verify encryption uses it
	onlyKey2 := &Keyring{}
	onlyKey2.AddKey(key2)

	// Should decrypt successfully with only key2
	decrypted2, err := onlyKey2.Decrypt(encrypted2, nil)
	if err != nil {
		t.Fatalf("Decrypt with only key2 failed (encryption should use new primary): %v", err)
	}

	if !bytes.Equal(decrypted2, data2) {
		t.Fatal("decrypted data2 does not match")
	}

	// Verify data1 still decrypts with both keys
	decrypted1, err := k.Decrypt(encrypted1, nil)
	if err != nil {
		t.Fatalf("Decrypt old data after rotation failed: %v", err)
	}
	if !bytes.Equal(decrypted1, data1) {
		t.Fatal("decrypted data1 does not match")
	}
}

// 23. Decrypt tries keys in order until success
func TestKeyring_Decrypt_TriesKeysInOrder(t *testing.T) {
	k := &Keyring{}

	// Create three separate keyrings for three keys
	k1 := &Keyring{}
	key1 := generateKey(32)
	k1.AddKey(key1)

	k2 := &Keyring{}
	key2 := generateKey(32)
	k2.AddKey(key2)

	k3 := &Keyring{}
	key3 := generateKey(32)
	k3.AddKey(key3)

	// Encrypt data with key2
	data := []byte("data encrypted with key2")
	encrypted, err := k2.Encrypt(data, nil)
	if err != nil {
		t.Fatalf("Encrypt with key2 failed: %v", err)
	}

	// Build main keyring: key3 (primary), key1, key2
	k.AddKey(key3)
	k.AddKey(key1)
	k.AddKey(key2)

	// Decrypt should succeed by trying key3, key1 (fail), then key2 (success)
	decrypted, err := k.Decrypt(encrypted, nil)
	if err != nil {
		t.Fatalf("Decrypt should have succeeded with key2: %v", err)
	}

	if !bytes.Equal(decrypted, data) {
		t.Fatal("decrypted data does not match")
	}
}

// 24. Keyring with keys that can't decrypt any (returns ErrKeyringCannotDecrypt)
func TestKeyring_Decrypt_CannotDecryptAny_ReturnsError(t *testing.T) {
	k := &Keyring{}

	// Add keys that won't match the encrypted data
	k.AddKey(generateKey(32))
	k.AddKey(generateKey(32))
	k.AddKey(generateKey(32))

	// Encrypt with a completely different key
	otherKey := &Keyring{}
	otherKey.AddKey(generateKey(32))

	data := []byte("unreachable data")
	encrypted, err := otherKey.Encrypt(data, nil)
	if err != nil {
		t.Fatalf("Encrypt with other key failed: %v", err)
	}

	_, err = k.Decrypt(encrypted, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrKeyringCannotDecrypt {
		t.Fatalf("expected ErrKeyringCannotDecrypt, got %v", err)
	}
}
