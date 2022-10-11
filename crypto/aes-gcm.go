package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// Encrypt is used to encrypt a data with a given key.
func Encrypt(secret, data, aad []byte) (encrypted []byte, err error) {
	var block cipher.Block
	if block, err = aes.NewCipher(secret); err != nil {
		return
	}

	var gcm cipher.AEAD
	if gcm, err = cipher.NewGCM(block); err != nil {
		return
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}

	encrypted = gcm.Seal(nonce, nonce, data, aad)

	return
}

// Decrypt is used to decrypt a message with a given key, and verify it's contents.
func Decrypt(secret, encrypted, aad []byte) (plain []byte, err error) {
	// Ensure we have at least one byte
	if len(encrypted) == 0 {
		return nil, fmt.Errorf("cannot decrypt empty payload")
	}

	var block cipher.Block
	if block, err = aes.NewCipher(secret); err != nil {
		return
	}

	var gcm cipher.AEAD
	if gcm, err = cipher.NewGCM(block); err != nil {
		return
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		err = ErrInvalidMessage
		return
	}

	nonce, cipherText := encrypted[:nonceSize], encrypted[nonceSize:]
	if plain, err = gcm.Open(nil, nonce, cipherText, aad); err != nil {
		return
	}

	return
}
