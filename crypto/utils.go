package crypto

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"hash"
)

var hmacSha2ToAlgoName = map[string][]byte{
	"sha256": []byte("HS256"),
	"sha384": []byte("HS384"),
	"sha512": []byte("HS512"),
}

var hmacSha2ToDigestType = map[string]string{
	"HS256": "sha256",
	"HS384": "sha384",
	"HS512": "sha512",
}

func getSha2Func(digest string) (func() hash.Hash, string) {
	switch digest {
	case "sha512":
		return sha512.New, "sha512"
	case "sha384":
		return sha512.New384, "sha384"
	default:
		return sha256.New, "sha256"
	}
}

// SecureBytesCompare Compares the two binaries in constant-time to avoid timing attacks.
//
// See: http://codahale.com/a-lesson-in-timing-attacks/
// Source: https://go.dev/play/p/pICufdp1zA
func SecureBytesCompare(input, secret []byte) bool {
	bs := [][]byte{input, secret}
	eq := subtle.ConstantTimeEq(int32(len(input)), int32(len(secret)))
	eq &= subtle.ConstantTimeCompare(bs[0], bs[eq])
	return eq == 1
}

var ErrKeySize = errors.New("key size must be 16, 24 or 32 bytes")

// ValidateKey will check to see if the key is valid and returns an error if not.
//
// key should be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func ValidateKey(key []byte) error {
	if l := len(key); l != 16 && l != 24 && l != 32 {
		return ErrKeySize
	}
	return nil
}
