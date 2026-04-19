// Package main demonstrates cryptographic utilities in Chain.
//
// Run with: go run main.go
//
// This example covers:
//   - AES-GCM encryption/decryption
//   - PBKDF2 key derivation
//   - Keyring with key rotation
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/nidorx/chain"
)

func main() {
	// ── Set up secret key ──────────────────────────────────────────────
	// In production, use a strong random key from environment or secrets manager
	secretKey := os.Getenv("SECRET_KEY_BASE")
	if secretKey == "" {
		// Generate a random 32-byte key for demo
		key := make([]byte, 32)
		rand.Read(key)
		secretKey = string(key)
		fmt.Println("[INFO] Using generated demo key")
	}

	if err := chain.SetSecretKeyBase(secretKey); err != nil {
		log.Fatalf("Failed to set secret key: %v", err)
	}
	fmt.Println("[OK] Secret key configured")

	// ── 1. AES-GCM Encryption ─────────────────────────────────────────
	fmt.Println("\n=== AES-GCM Encryption ===")

	plaintext := []byte("This is a secret message")
	aad := []byte("additional authenticated data") // optional context

	// Encrypt
	encrypted, err := chain.Crypto().Encrypt([]byte(secretKey), plaintext, aad)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	fmt.Printf("Plaintext:  %s\n", plaintext)
	fmt.Printf("Encrypted:  %s (base64)\n", base64.StdEncoding.EncodeToString(encrypted))

	// Decrypt
	decrypted, err := chain.Crypto().Decrypt([]byte(secretKey), encrypted, aad)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}
	fmt.Printf("Decrypted:  %s\n", decrypted)
	fmt.Println("[OK] AES-GCM encrypt/decrypt works!")

	// ── 2. Key Derivation (PBKDF2) ─────────────────────────────────────
	fmt.Println("\n=== Key Derivation (PBKDF2) ===")

	password := []byte("my-secure-password")
	salt := []byte("unique-salt-value")

	// Derive keys with recommended parameters
	key1 := chain.Crypto().KeyGenerate(password, salt, 216000, 32, "sha256")
	key2 := chain.Crypto().KeyGenerate(password, salt, 216000, 32, "sha256")

	fmt.Printf("Derived key 1: %s (base64)\n", base64.StdEncoding.EncodeToString(key1))
	fmt.Printf("Derived key 2: %s (base64)\n", base64.StdEncoding.EncodeToString(key2))

	// Same password + salt = same key (deterministic)
	if string(key1) == string(key2) {
		fmt.Println("[OK] Key derivation is deterministic")
	}

	// Different iterations = different key
	key3 := chain.Crypto().KeyGenerate(password, salt, 1000, 32, "sha256")
	fmt.Printf("Different iterations: %s (base64)\n", base64.StdEncoding.EncodeToString(key3))
	if string(key1) != string(key3) {
		fmt.Println("[OK] Different parameters produce different keys")
	}

	// ── 3. Message Signing (HMAC) ──────────────────────────────────────
	fmt.Println("\n=== Message Signing ===")

	message := []byte("important data")

	// Sign
	signature := chain.Crypto().MessageSign([]byte(secretKey), message, "sha256")
	fmt.Printf("Message:    %s\n", message)
	fmt.Printf("Signature:  %s\n", signature[:50]+"...")

	// Verify
	decoded, err := chain.Crypto().MessageVerify([]byte(secretKey), []byte(signature))
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}
	fmt.Printf("Decoded:    %s\n", decoded)
	fmt.Println("[OK] Message signing/verification works!")

	// ── 4. Message Encryption (Authenticated) ──────────────────────────
	fmt.Println("\n=== Message Encryption (Authenticated) ===")

	content := []byte("confidential information")

	// Encrypt with authentication
	encoded, err := chain.Crypto().MessageEncrypt([]byte(secretKey), content, aad)
	if err != nil {
		log.Fatalf("Message encryption failed: %v", err)
	}
	fmt.Printf("Content:    %s\n", content)
	fmt.Printf("Encrypted:  %s\n", encoded[:50]+"...")

	// Decrypt and verify
	decryptedContent, err := chain.Crypto().MessageDecrypt([]byte(secretKey), []byte(encoded), aad)
	if err != nil {
		log.Fatalf("Message decryption failed: %v", err)
	}
	fmt.Printf("Decrypted:  %s\n", decryptedContent)
	fmt.Println("[OK] Message encrypt/decrypt works!")

	// ── 5. Keyring (Key Rotation) ──────────────────────────────────────
	fmt.Println("\n=== Keyring (Key Rotation) ===")

	// Create a keyring that syncs with SecretKeyBase
	keyring := chain.NewKeyring("keyring-salt", 216000, 32, "sha256")

	// Encrypt with the primary key
	data := []byte("data encrypted with primary key")
	encryptedData, err := keyring.Encrypt(data, nil)
	if err != nil {
		log.Fatalf("Keyring encryption failed: %v", err)
	}
	fmt.Printf("Data:        %s\n", data)
	fmt.Printf("Encrypted:   %s (base64)\n", base64.StdEncoding.EncodeToString(encryptedData))

	// Decrypt (tries all keys in the ring)
	decryptedData, err := keyring.Decrypt(encryptedData, nil)
	if err != nil {
		log.Fatalf("Keyring decryption failed: %v", err)
	}
	fmt.Printf("Decrypted:   %s\n", decryptedData)
	fmt.Println("[OK] Keyring encrypt/decrypt works!")

	// Simulate key rotation: add a new key (becomes primary)
	fmt.Println("\n--- Simulating key rotation ---")
	newKey := make([]byte, 32)
	for i := range newKey {
		newKey[i] = byte(i + 100)
	}
	keyring.AddKey(newKey)

	// Old data can still be decrypted with the old key
	decryptedOldData, err := keyring.Decrypt(encryptedData, nil)
	if err != nil {
		log.Fatalf("Failed to decrypt old data after rotation: %v", err)
	}
	fmt.Printf("Old data decrypted: %s\n", decryptedOldData)
	fmt.Println("[OK] Old data still decryptable after rotation!")

	// New data encrypted with the new primary key
	newData := []byte("data after rotation")
	newEncrypted, _ := keyring.Encrypt(newData, nil)
	newDecrypted, _ := keyring.Decrypt(newEncrypted, nil)
	fmt.Printf("New data: %s -> %s\n", newData, newDecrypted)
	fmt.Println("[OK] Keyring rotation complete!")

	fmt.Println("\n=== All crypto demos passed! ===")
}
