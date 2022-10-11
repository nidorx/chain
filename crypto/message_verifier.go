// Copyright 2022 Alex Rodin. All rights reserved.
// Based on the https://github.com/elixir-plug/plug_crypto package, Copyright (c) 2018 Plataformatec.

package crypto

import (
	"bytes"
	"crypto/hmac"
	"encoding/base64"
	"errors"
)

// base64 without padding
var b64NoPad = base64.RawURLEncoding

var ErrInvalidSignature = errors.New("invalid signature")

// MessageVerifier makes it easy to generate and verify messages which are signed to prevent tampering.
type MessageVerifier struct {
}

// Sign Generates a signed message for the provided value.
//
// See https://www.rfc-editor.org/rfc/rfc7515#section-3.2
// See https://www.rfc-editor.org/rfc/rfc7515#section-7
func (v *MessageVerifier) Sign(secret []byte, content []byte, digest string) string {
	sha2Func, digest := getSha2Func(digest)
	algo := hmacSha2ToAlgoName[digest]

	// <Header>.<Payload>
	protected := b64NoPad.EncodeToString(algo)
	payload := b64NoPad.EncodeToString(content)
	signingString := protected + "." + payload

	hash := hmac.New(sha2Func, secret)
	hash.Write([]byte(signingString))
	signature := hash.Sum(nil)

	// <Header>.<Payload>.<Signature>
	return signingString + "." + b64NoPad.EncodeToString(signature)
}

// Verify Decodes and verifies the encoded binary was not tampered with.
func (v *MessageVerifier) Verify(secret []byte, signed []byte) (decoded []byte, err error) {
	var (
		algo          []byte
		payload       []byte
		signingString []byte
		signature     []byte
	)
	if algo, payload, signingString, signature, err = v.decodeToken(signed); err != nil {
		return
	}
	sha2Func, _ := getSha2Func(hmacSha2ToDigestType[string(algo)])
	hash := hmac.New(sha2Func, secret)
	hash.Write(signingString)
	challenge := hash.Sum(nil)

	if SecureBytesCompare(challenge, signature) {
		decoded = payload
	} else {
		err = ErrInvalidSignature
	}
	return
}

// decodeToken base64.Decode(token.split(".", 3))
func (v *MessageVerifier) decodeToken(token []byte) (
	algo []byte, payload []byte, plainText []byte, signature []byte, err error,
) {

	var (
		algo64    []byte
		payload64 []byte
	)

	// algo name
	rest := token[0:]
	index := bytes.IndexByte(rest, '.')
	algo64 = rest[0:index]

	rest = rest[index+1:]
	index = bytes.IndexByte(rest, '.')
	payload64 = rest[0:index]

	plainText = make([]byte, len(algo64)+len(payload64)+1)
	copy(plainText[0:], algo64)
	plainText[len(algo64)] = '.'
	copy(plainText[len(algo64)+1:], payload64)

	// decode algo name
	algo = make([]byte, b64NoPad.DecodedLen(len(algo64)))
	if _, err = b64NoPad.Decode(algo, algo64); err != nil {
		return
	}

	// decode payload
	payload = make([]byte, b64NoPad.DecodedLen(len(payload64)))
	if _, err = b64NoPad.Decode(payload, payload64); err != nil {
		return
	}

	// signature
	rest = rest[index+1:]
	signature = make([]byte, b64NoPad.DecodedLen(len(rest)))
	if _, err = b64NoPad.Decode(signature, rest); err != nil {
		return
	}

	return
}
