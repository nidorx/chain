package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var (
	A128GCM           = []byte("A128GCM")
	ErrInvalidMessage = errors.New("invalid message")
)

const (
	authTagLength = 16
)

// MessageEncryptor is a simple way to encrypt values which get stored somewhere you don't trust.
//
// The encrypted key, initialization vector, cipher text, and cipher tag are base64url encoded and returned to you.
//
// This can be used in situations similar to the `MessageVerifier`, but where you don't want users to be able to
// determine the value of the payload.
//
// The current algorithm used is AES-GCM-128.
type MessageEncryptor struct {
}

// Encrypt a message using authenticated encryption.
func (e *MessageEncryptor) Encrypt(message []byte, secret []byte, signSecret []byte) (encrypted string, err error) {
	return e.aes128GCMEncrypt(message, secret, signSecret)
}

// Decrypt a message using authenticated encryption.
func (e *MessageEncryptor) Decrypt(encrypted []byte, secret []byte, signSecret []byte) ([]byte, error) {
	return e.aes128GCMDecrypt(encrypted, secret, signSecret)
}

// Encrypts and authenticates a message using AES128-GCM mode.
//
// A random 128-bit content encryption key (CEK) is generated for every message which is then encrypted with
// aesGCMKeyWrap.
func (e *MessageEncryptor) aes128GCMEncrypt(plainText []byte, secret []byte, signSecret []byte) (encrypted string, err error) {
	cek := make([]byte, 16) // a 128-bit content encryption key (CEK)
	if _, err = io.ReadFull(rand.Reader, cek); err != nil {
		return
	}

	var (
		cipherText   []byte // an encrypted cipher text of the same length as the original string
		encryptedCEK []byte //
	)

	// encripta o conteúdo com o CEK
	if cipherText, err = e.blockEncrypt(cek, plainText, A128GCM); err != nil {
		return
	}

	// encripta o CEK com a secret
	if encryptedCEK, err = e.aesGCMKeyWrap(cek, secret, signSecret); err != nil {
		return
	}

	encrypted = e.encodeToken(A128GCM, encryptedCEK, cipherText)
	return
}

// Verifies and decrypts a message using AES128-GCM mode.
//
// Decryption will never be performed prior to verification.
//
// The encrypted content encryption key (CEK) is decrypted with aesGCMKeyUnwrap.
func (e *MessageEncryptor) aes128GCMDecrypt(encoded, secret, signSecret []byte) (decrypted []byte, err error) {
	var (
		aad          []byte // additional authenticated data
		encryptedCEK []byte // a 128-bit content encryption key (CEK)
		cipherText   []byte
	)
	if aad, encryptedCEK, cipherText, err = e.decodeToken(encoded); err != nil {
		return
	}

	var cek []byte
	if cek, err = e.aesGCMKeyUnwrap(encryptedCEK, secret, signSecret); err != nil {
		return
	}

	// decripta o conteúdo usando o CEK
	decrypted, err = e.blockDecrypt(cek, cipherText, aad)
	return
}

// aesGCMKeyWrap Wraps a decrypted content encryption key (CEK) with secret and signSecret using AES GCM mode.
//
// Accepts keys of 128, 192, or  256 bits based on the length of the secret key.
//
// See: https://tools.ietf.org/html/rfc7518#section-4.7
func (e *MessageEncryptor) aesGCMKeyWrap(cek, secret, signSecret []byte) (encryptedCEK []byte, err error) {
	if len(secret) > 32 {
		// bit_size(secret) > 256
		secret = secret[:32]
	}

	if encryptedCEK, err = e.blockEncrypt(secret, cek, signSecret); err != nil {
		return
	}
	return
}

// aesGCMKeyUnwrap Unwraps an encrypted content encryption key (CEK) with secret and signSecret using AES GCM mode.
//
// Accepts keys of 128, 192, or 256  bits based on the length of the secret key.
//
// See: https://tools.ietf.org/html/rfc7518#section-4.7
func (e *MessageEncryptor) aesGCMKeyUnwrap(encryptedCEK, secret, signSecret []byte) (cek []byte, err error) {
	if len(secret) > 32 {
		// bit_size(secret) > 256
		secret = secret[:32]
	}
	cek, err = e.blockDecrypt(secret, encryptedCEK, signSecret)
	return
}

// blockEncrypt
func (e *MessageEncryptor) blockEncrypt(key, data, aad []byte) (cipherText []byte, err error) {
	var block cipher.Block
	if block, err = aes.NewCipher(key); err != nil {
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

	cipherText = gcm.Seal(nonce, nonce, data, aad)

	return
}

// blockEncrypt
func (e *MessageEncryptor) blockDecrypt(key, data, aad []byte) (plainContent []byte, err error) {
	var block cipher.Block
	if block, err = aes.NewCipher(key); err != nil {
		return
	}

	var gcm cipher.AEAD
	if gcm, err = cipher.NewGCM(block); err != nil {
		return
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		err = ErrInvalidMessage
		return
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	if plainContent, err = gcm.Open(nil, nonce, cipherText, aad); err != nil {
		return
	}

	return
}

func (e *MessageEncryptor) encodeToken(aad, encryptedKey, cipherText []byte) string {
	return b64NoPad.EncodeToString(aad) +
		"." +
		b64NoPad.EncodeToString(encryptedKey) +
		"." +
		b64NoPad.EncodeToString(cipherText)
}

// decodeToken base64.Decode(token.split(".", 5))
func (e *MessageEncryptor) decodeToken(token []byte) (
	aad []byte, encryptedKey []byte, cipherText []byte, err error,
) {
	// aad.encryptedKey.cipherText

	// aad
	rest := token[0:]
	index := bytes.IndexByte(rest, '.')
	aad = make([]byte, b64NoPad.DecodedLen(index))
	if _, err = b64NoPad.Decode(aad, rest[0:index]); err != nil {
		return
	}

	// encrypted key
	rest = rest[index+1:]
	index = bytes.IndexByte(rest, '.')
	encryptedKey = make([]byte, b64NoPad.DecodedLen(index))
	if _, err = b64NoPad.Decode(encryptedKey, rest[0:index]); err != nil {
		return
	}

	// signature
	rest = rest[index+1:]
	cipherText = make([]byte, b64NoPad.DecodedLen(len(rest)))
	if _, err = b64NoPad.Decode(cipherText, rest); err != nil {
		return
	}

	return
}
