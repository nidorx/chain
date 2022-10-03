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
func (v *MessageVerifier) Sign(message []byte, secret []byte, digest string) string {
	sha2Func, digest := getSha2Func(digest)
	algoName := hmacSha2ToAlgoName[digest]
	plainText := v.signingInput(algoName, message)
	signature := hmac.New(sha2Func, secret).Sum([]byte(plainText))
	return v.encodeToken(plainText, signature)
}

// Verify Decodes and verifies the encoded binary was not tampered with.
func (v *MessageVerifier) Verify(signed []byte, secret []byte) (decoded []byte, err error) {
	var (
		algo      []byte
		payload   []byte
		plainText []byte
		signature []byte
	)
	if algo, payload, plainText, signature, err = v.decodeToken(signed); err != nil {
		return
	}
	sha2Func, _ := getSha2Func(hmacSha2ToDigestType[string(algo)])
	// signature := hmac.New(sha2Func, secret).Sum([]byte(plainText))
	challenge := hmac.New(sha2Func, secret).Sum(plainText)

	if SecureBytesCompare(challenge, signature) {
		decoded = payload
	} else {
		err = ErrInvalidSignature
	}
	return
}

func (v *MessageVerifier) signingInput(protected []byte, payload []byte) string {

	return b64NoPad.EncodeToString(protected) + "." + b64NoPad.EncodeToString(payload)
}

func (v *MessageVerifier) encodeToken(plainText string, signature []byte) string {
	// base64 without padding
	return plainText + "." + b64NoPad.EncodeToString(signature)
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