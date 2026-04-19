// Package main demonstrates message signing and verification in Chain.
//
// This example shows how to create and verify tamper-proof tokens,
// similar to JWT but using HMAC signatures.
//
// Run with: go run main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nidorx/chain"
)

// TokenPayload represents the data we want to sign.
type TokenPayload struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

func main() {
	// ── Set up secret key ──────────────────────────────────────────────
	if err := chain.SetSecretKeyBase("example-secret-key-for-signing!!"); err != nil {
		log.Fatalf("Failed to set secret key: %v", err)
	}
	secretKey := []byte(chain.SecretKeyBase())

	fmt.Println("=== Message Signing & Verification ===")

	// ── 1. Create and sign a token ─────────────────────────────────────
	fmt.Println("--- Creating a signed token ---")

	payload := TokenPayload{
		UserID:    "user-123",
		Username:  "alice",
		Role:      "admin",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IssuedAt:  time.Now(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal payload: %v", err)
	}

	// Sign the payload
	signature := chain.Crypto().MessageSign(secretKey, payloadJSON, "sha256")
	fmt.Printf("Payload:  %s\n", payloadJSON)
	fmt.Printf("Token:    %s\n", signature)
	fmt.Println("[OK] Token signed successfully")

	// ── 2. Verify the token ────────────────────────────────────────────
	fmt.Println("\n--- Verifying the token ---")

	decoded, err := chain.Crypto().MessageVerify(secretKey, []byte(signature))
	if err != nil {
		log.Fatalf("Token verification failed: %v", err)
	}

	var verifiedPayload TokenPayload
	if err := json.Unmarshal(decoded, &verifiedPayload); err != nil {
		log.Fatalf("Failed to unmarshal payload: %v", err)
	}

	fmt.Printf("Verified user: %s (ID: %s, Role: %s)\n",
		verifiedPayload.Username, verifiedPayload.UserID, verifiedPayload.Role)
	fmt.Printf("Token expires: %s\n", verifiedPayload.ExpiresAt.Format(time.RFC3339))
	fmt.Println("[OK] Token verified successfully")

	// ── 3. Tamper detection ────────────────────────────────────────────
	fmt.Println("\n--- Tamper detection ---")

	// Simulate a tampered token (modify a character)
	tampered := signature[:10] + "X" + signature[11:]
	fmt.Printf("Tampered:  %s\n", tampered)

	_, err = chain.Crypto().MessageVerify(secretKey, []byte(tampered))
	if err != nil {
		fmt.Printf("[OK] Tampering detected! Error: %v\n", err)
	} else {
		fmt.Println("[FAIL] Tampering was NOT detected!")
	}

	// ── 4. Using wrong key ─────────────────────────────────────────────
	fmt.Println("\n--- Wrong key verification ---")

	wrongKey := []byte("wrong-secret-key!!")
	_, err = chain.Crypto().MessageVerify(wrongKey, []byte(signature))
	if err != nil {
		fmt.Printf("[OK] Wrong key rejected! Error: %v\n", err)
	} else {
		fmt.Println("[FAIL] Wrong key was accepted!")
	}

	// ── 5. Token expiration check ──────────────────────────────────────
	fmt.Println("\n--- Token expiration ---")

	// Create an expired token
	expiredPayload := TokenPayload{
		UserID:    "user-456",
		Username:  "bob",
		Role:      "user",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired 1 hour ago
		IssuedAt:  time.Now().Add(-2 * time.Hour),
	}

	expiredJSON, _ := json.Marshal(expiredPayload)
	expiredSignature := chain.Crypto().MessageSign(secretKey, expiredJSON, "sha256")

	// Verify the signature (signature is valid, but payload is expired)
	expiredDecoded, err := chain.Crypto().MessageVerify(secretKey, []byte(expiredSignature))
	if err != nil {
		log.Fatalf("Expired token verification failed: %v", err)
	}

	var expiredData TokenPayload
	json.Unmarshal(expiredDecoded, &expiredData)

	if time.Now().After(expiredData.ExpiresAt) {
		fmt.Printf("[OK] Token signature valid, but expired! User: %s\n", expiredData.Username)
	}

	// ── 6. Practical: API token flow ───────────────────────────────────
	fmt.Println("\n--- Practical API Token Flow ---")

	// Server: Generate a token for a user after login
	loginToken := generateAuthToken("user-789", "charlie", "user", secretKey)
	fmt.Printf("Login token generated: %s...\n", loginToken[:40])

	// Client: Sends token in Authorization header (simulated)
	// Server: Verifies the token on subsequent requests
	user, err := verifyAuthToken(loginToken, secretKey)
	if err != nil {
		log.Fatalf("Auth token verification failed: %v", err)
	}
	fmt.Printf("Authenticated user: %s (role: %s)\n", user.Username, user.Role)
	fmt.Println("[OK] API token flow works!")

	fmt.Println("\n=== All signing/verification demos passed! ===")
}

// generateAuthToken creates a signed token for a user.
func generateAuthToken(userID, username, role string, secretKey []byte) string {
	payload := TokenPayload{
		UserID:    userID,
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IssuedAt:  time.Now(),
	}

	payloadJSON, _ := json.Marshal(payload)
	return chain.Crypto().MessageSign(secretKey, payloadJSON, "sha256")
}

// verifyAuthToken verifies a token and returns the payload.
func verifyAuthToken(token string, secretKey []byte) (*TokenPayload, error) {
	decoded, err := chain.Crypto().MessageVerify(secretKey, []byte(token))
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	var payload TokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Check expiration
	if time.Now().After(payload.ExpiresAt) {
		return nil, fmt.Errorf("token expired at %s", payload.ExpiresAt.Format(time.RFC3339))
	}

	return &payload, nil
}
