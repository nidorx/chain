// Copyright 2022 Alex Rodin. All rights reserved.
// Based on the https://github.com/elixir-plug/plug_crypto package, Copyright (c) 2018 Plataformatec.

package crypto

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"strings"
)

var (
	HEADER            = []byte("A128GCM")
	ErrInvalidMessage = errors.New("invalid message")
)

// MessageEncryptor is a simple way to encrypt values which get stored somewhere you don't trust.
//
// This can be used in situations similar to the `MessageVerifier`, but where you don't want users to be able to
// determine the value of the payload.
//
// The current algorithm used is AES-GCM-128.

type MessageEncryptor struct {
}

// Encrypt encrypts and authenticates a message using AES128-GCM mode.
//
// A random 128-bit content encryption key (CEK) is generated for every message which is then encrypted with secret and
// aad using AES GCM mode.
//
// See: https://tools.ietf.org/html/rfc7518#section-4.7
func (e *MessageEncryptor) Encrypt(secret, content, aad []byte) (encoded string, err error) {
	cek := make([]byte, 16) // a 128-bit content encryption key (CEK)
	if _, err = io.ReadFull(rand.Reader, cek); err != nil {
		return
	}

	var (
		encryptedCEK     []byte
		encryptedContent []byte
	)

	// encrypts message with CEK
	if encryptedContent, err = Encrypt(cek, content, HEADER); err != nil {
		return
	}

	if len(secret) > 32 {
		// bit_size(secret) > 256
		secret = secret[:32]
	}

	// encrypt the CEK with the secret
	//
	// wraps a decrypted content encryption key (CEK) with secret and aad using AES GCM mode.
	if encryptedCEK, err = Encrypt(secret, cek, aad); err != nil {
		return
	}

	// encode token
	// <Header>.<Encrypted_Key>.<Encrypted_Content>
	encoded = strings.Join([]string{
		b64NoPad.EncodeToString(HEADER),
		b64NoPad.EncodeToString(encryptedCEK),
		b64NoPad.EncodeToString(encryptedContent),
	}, ".")
	return
}

// Decrypt a message using authenticated encryption.
// Accepts keys of 128, 192, or  256 bits based on the length of the secret key.
// Verifies and decrypts a message using AES128-GCM mode.
//
// Decryption will never be performed prior to verification.
//
// The encrypted content encryption key (CEK) is decrypted with aesGCMKeyUnwrap.
func (e *MessageEncryptor) Decrypt(secret, encoded, aad []byte) (content []byte, err error) {
	var (
		header           []byte
		encryptedCEK     []byte
		encryptedContent []byte
		cek              []byte
	)
	// <Header>.<Encrypted_Key>.<Encrypted_Content>
	if header, encryptedCEK, encryptedContent, err = e.decodeToken(encoded); err != nil {
		return
	}

	if len(secret) > 32 {
		// bit_size(secret) > 256
		secret = secret[:32]
	}

	// decrypt the CEK with the secret
	if cek, err = Decrypt(secret, encryptedCEK, aad); err != nil {
		return
	}

	// decrypt content using CEK
	content, err = Decrypt(cek, encryptedContent, header)
	return
}

// decodeToken base64.Decode(token.split(".", 3))
func (e *MessageEncryptor) decodeToken(token []byte) (
	aadA128GCM []byte, encryptedCEK []byte, encryptedContent []byte, err error,
) {
	// aad
	rest := token[0:]
	index := bytes.IndexByte(rest, '.')
	aadA128GCM = make([]byte, b64NoPad.DecodedLen(index))
	if _, err = b64NoPad.Decode(aadA128GCM, rest[0:index]); err != nil {
		return
	}

	// encrypted key
	rest = rest[index+1:]
	index = bytes.IndexByte(rest, '.')
	encryptedCEK = make([]byte, b64NoPad.DecodedLen(index))
	if _, err = b64NoPad.Decode(encryptedCEK, rest[0:index]); err != nil {
		return
	}

	// encrypted content
	rest = rest[index+1:]
	encryptedContent = make([]byte, b64NoPad.DecodedLen(len(rest)))
	if _, err = b64NoPad.Decode(encryptedContent, rest); err != nil {
		return
	}

	return
}
