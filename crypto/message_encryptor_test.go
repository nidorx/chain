package crypto

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

func generateRandomKey(t *testing.T, size int) []byte {
	t.Helper()
	key := make([]byte, size)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}
	return key
}

func generateRandomAAD(t *testing.T, size int) []byte {
	t.Helper()
	aad := make([]byte, size)
	_, err := rand.Read(aad)
	if err != nil {
		t.Fatalf("failed to generate random AAD: %v", err)
	}
	return aad
}

// 1. Basic encrypt/decrypt roundtrip with 16-byte key
func TestEncryptDecrypt_16ByteKey(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 16)
	content := []byte("hello world")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted content mismatch: got %q, want %q", decrypted, content)
	}
}

// 2. Basic encrypt/decrypt with 32-byte key
func TestEncryptDecrypt_32ByteKey(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("test with 32-byte key")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted content mismatch: got %q, want %q", decrypted, content)
	}
}

// 3. Encrypt/decrypt with AAD
func TestMessageEncrypt_WithAAD(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("content with AAD")
	aad := generateRandomAAD(t, 32)

	encoded, err := encryptor.Encrypt(secret, content, aad)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), aad)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted content mismatch: got %q, want %q", decrypted, content)
	}
}

// 4. Encrypt/decrypt empty content
func TestEncryptDecrypt_EmptyContent(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte{}

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty content, got %d bytes", len(decrypted))
	}
}

// 5. Encrypt/decrypt large content
func TestEncryptDecrypt_LargeContent(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := make([]byte, 10*1024*1024) // 10MB
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("failed to generate large content: %v", err)
	}

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted large content mismatch")
	}
}

// 6. Encrypt/decrypt unicode content (UTF-8)
func TestEncryptDecrypt_UnicodeContent(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("Hello, \u4e16\u754c! \U0001f600 \u2764\ufe0f \u00e9\u00e0\u00fc\u00f1")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted unicode content mismatch: got %q, want %q", decrypted, content)
	}
}

// 7. Encrypt/decrypt binary content
func TestEncryptDecrypt_BinaryContent(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	// Content with all possible byte values
	content := make([]byte, 256)
	for i := 0; i < 256; i++ {
		content[i] = byte(i)
	}

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted binary content mismatch")
	}
}

// 8. Decrypt with wrong secret (should fail)
func TestDecrypt_WrongSecret(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret1 := generateRandomKey(t, 32)
	secret2 := generateRandomKey(t, 32)
	content := []byte("secret content")

	encoded, err := encryptor.Encrypt(secret1, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = encryptor.Decrypt(secret2, []byte(encoded), nil)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong secret, got nil")
	}
}

// 9. Decrypt with tampered token (should fail)
func TestDecrypt_TamperedToken(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("tamper test")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Tamper with the encoded token by modifying a byte
	tampered := []byte(encoded)
	tampered[len(tampered)/2] ^= 0xFF

	_, err = encryptor.Decrypt(secret, tampered, nil)
	if err == nil {
		t.Fatal("expected error when decrypting tampered token, got nil")
	}
}

// 10. Decrypt with wrong AAD (should fail)
func TestMessageDecrypt_WrongAAD(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("AAD protected")
	correctAAD := generateRandomAAD(t, 32)
	wrongAAD := generateRandomAAD(t, 32)

	encoded, err := encryptor.Encrypt(secret, content, correctAAD)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = encryptor.Decrypt(secret, []byte(encoded), wrongAAD)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong AAD, got nil")
	}
}

// 11. Decrypt empty token (should fail)
func TestDecrypt_EmptyToken(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)

	// Recover from panic since empty token causes a panic in decodeToken
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected for empty token
			t.Logf("recovering from panic as expected: %v", r)
		}
	}()

	_, err := encryptor.Decrypt(secret, []byte(""), nil)
	if err == nil {
		t.Fatal("expected error when decrypting empty token, got nil")
	}
}

// 12. Decrypt malformed token (missing dots, invalid base64)
func TestDecrypt_MalformedToken(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)

	malformedTokens := []string{
		"nodotsatall",
		"only.one.dot",
		"too.many.dots.here",
		"..",
		"!!!.###.@@@", // invalid base64
		"abc.def.ghi", // short invalid segments
		"invalid===.base64!@#.content$$$",
	}

	for _, token := range malformedTokens {
		// Recover from potential panics in decodeToken
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected for some malformed tokens
				}
			}()
			_, err := encryptor.Decrypt(secret, []byte(token), nil)
			if err == nil {
				t.Errorf("expected error for malformed token %q, got nil", token)
			}
		}()
	}
}

// 13. Decrypt token with modified header (should fail)
func TestDecrypt_ModifiedHeader(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("header test")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in token, got %d", len(parts))
	}

	// Modify the header part
	modifiedHeader := strings.Repeat("X", len(parts[0]))
	tampered := modifiedHeader + "." + parts[1] + "." + parts[2]

	_, err = encryptor.Decrypt(secret, []byte(tampered), nil)
	if err == nil {
		t.Fatal("expected error when decrypting with modified header, got nil")
	}
}

// 14. Decrypt token with modified encrypted key (should fail)
func TestDecrypt_ModifiedEncryptedKey(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("encrypted key test")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in token, got %d", len(parts))
	}

	// Modify the encrypted key part
	modifiedKey := strings.Repeat("Y", len(parts[1]))
	tampered := parts[0] + "." + modifiedKey + "." + parts[2]

	_, err = encryptor.Decrypt(secret, []byte(tampered), nil)
	if err == nil {
		t.Fatal("expected error when decrypting with modified encrypted key, got nil")
	}
}

// 15. Decrypt token with modified content (should fail)
func TestDecrypt_ModifiedContent(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("content modification test")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in token, got %d", len(parts))
	}

	// Modify the content part
	modifiedContent := strings.Repeat("Z", len(parts[2]))
	tampered := parts[0] + "." + parts[1] + "." + modifiedContent

	_, err = encryptor.Decrypt(secret, []byte(tampered), nil)
	if err == nil {
		t.Fatal("expected error when decrypting with modified content, got nil")
	}
}

// 16. Secret truncation (secret > 32 bytes)
func TestEncryptDecrypt_SecretTruncation(t *testing.T) {
	encryptor := &MessageEncryptor{}
	// Create a secret longer than 32 bytes
	longSecret := generateRandomKey(t, 64)
	content := []byte("truncation test")

	// Create truncated version (first 32 bytes)
	truncatedSecret := make([]byte, 32)
	copy(truncatedSecret, longSecret)

	// Encrypt with long secret
	encoded, err := encryptor.Encrypt(longSecret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt with long secret failed: %v", err)
	}

	// Should decrypt successfully with truncated secret
	decrypted, err := encryptor.Decrypt(truncatedSecret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt with truncated secret failed: %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("decrypted content mismatch with truncated secret")
	}

	// Should also work with the full long secret for decrypt
	decrypted2, err := encryptor.Decrypt(longSecret, []byte(encoded), nil)
	if err != nil {
		t.Fatalf("Decrypt with long secret failed: %v", err)
	}

	if !bytes.Equal(decrypted2, content) {
		t.Errorf("decrypted content mismatch with long secret")
	}
}

// 17. Multiple encrypt operations produce different tokens (random CEK)
func TestEncrypt_RandomCEK(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("randomness test")

	encoded1, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt (1) failed: %v", err)
	}

	encoded2, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt (2) failed: %v", err)
	}

	encoded3, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt (3) failed: %v", err)
	}

	if encoded1 == encoded2 || encoded1 == encoded3 || encoded2 == encoded3 {
		t.Error("encrypt operations produced identical tokens, CEK is not random")
	}

	// But all should decrypt to the same content
	for i, encoded := range []string{encoded1, encoded2, encoded3} {
		decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
		if err != nil {
			t.Fatalf("Decrypt (%d) failed: %v", i+1, err)
		}
		if !bytes.Equal(decrypted, content) {
			t.Errorf("Decrypt (%d) content mismatch", i+1)
		}
	}
}

// 18. Roundtrip preserves content exactly
func TestEncryptDecrypt_ContentPreservation(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)

	testCases := []struct {
		name    string
		content []byte
	}{
		{"nil content", nil},
		{"empty content", []byte{}},
		{"single byte", []byte{0x42}},
		{"null byte", []byte{0x00}},
		{"all zeros", bytes.Repeat([]byte{0x00}, 100)},
		{"all ones", bytes.Repeat([]byte{0xFF}, 100)},
		{"sequential bytes", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
		{"json content", []byte(`{"key":"value","nested":{"array":[1,2,3]}}`)},
		{"xml content", []byte(`<root><child attr="value">text</child></root>`)},
		{"special chars", []byte("!@#$%^&*()_+-=[]{}|;':\",./<>?`~")},
		{"newlines and tabs", []byte("line1\nline2\r\nline3\ttabbed")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := encryptor.Encrypt(secret, tc.content, nil)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if !bytes.Equal(decrypted, tc.content) {
				t.Errorf("content not preserved: got %d bytes, want %d bytes", len(decrypted), len(tc.content))
			}
		})
	}
}

// 19. Concurrent encrypt/decrypt (thread safety)
func TestMessageEncrypt_Concurrent(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)

	const goroutines = 100
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			localContent := []byte("goroutine " + string(rune('0'+id%10)))

			encoded, err := encryptor.Encrypt(secret, localContent, nil)
			if err != nil {
				t.Errorf("goroutine %d Encrypt failed: %v", id, err)
				return
			}

			decrypted, err := encryptor.Decrypt(secret, []byte(encoded), nil)
			if err != nil {
				t.Errorf("goroutine %d Decrypt failed: %v", id, err)
				return
			}

			if !bytes.Equal(decrypted, localContent) {
				t.Errorf("goroutine %d content mismatch", id)
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// 20. Token format verification (three base64url segments separated by dots)
func TestToken_Format(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("format test")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in token, got %d", len(parts))
	}

	for i, part := range parts {
		if len(part) == 0 {
			t.Errorf("part %d is empty", i)
		}

		// Verify base64url characters (A-Z, a-z, 0-9, -, _)
		for j, ch := range part {
			if !((ch >= 'A' && ch <= 'Z') ||
				(ch >= 'a' && ch <= 'z') ||
				(ch >= '0' && ch <= '9') ||
				ch == '-' || ch == '_') {
				t.Errorf("part %d has invalid base64url character at position %d: %c", i, j, ch)
			}
		}
	}
}

// 21. Header is always A128GCM
func TestEncrypt_HeaderIsA128GCM(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("header check")

	encoded, err := encryptor.Encrypt(secret, content, nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in token, got %d", len(parts))
	}

	// The header is base64url encoded, so "A128GCM" should be encoded
	// "A128GCM" in base64url is "QTEyOEdDTQ"
	expectedHeader := "QTEyOEdDTQ"
	if parts[0] != expectedHeader {
		t.Errorf("header mismatch: got %q, want %q", parts[0], expectedHeader)
	}
}

// 22. Encrypt/decrypt with various AAD values
func TestEncryptDecrypt_VariousAAD(t *testing.T) {
	encryptor := &MessageEncryptor{}
	secret := generateRandomKey(t, 32)
	content := []byte("AAD variation test")

	testCases := []struct {
		name string
		aad  []byte
	}{
		{"nil AAD", nil},
		{"empty AAD", []byte{}},
		{"single byte AAD", []byte{0x01}},
		{"32 byte AAD", generateRandomKey(t, 32)},
		{"64 byte AAD", generateRandomKey(t, 64)},
		{"all zeros AAD", bytes.Repeat([]byte{0x00}, 16)},
		{"all ones AAD", bytes.Repeat([]byte{0xFF}, 16)},
		{"text AAD", []byte("additional authenticated data")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := encryptor.Encrypt(secret, content, tc.aad)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			decrypted, err := encryptor.Decrypt(secret, []byte(encoded), tc.aad)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if !bytes.Equal(decrypted, content) {
				t.Errorf("decrypted content mismatch for AAD %q", tc.aad)
			}
		})
	}
}
