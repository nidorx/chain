// Copyright 2022 Alex Rodin. All rights reserved.
// Based on the https://github.com/elixir-plug/plug_crypto package, Copyright (c) 2018 Plataformatec.

package crypto

import "golang.org/x/crypto/pbkdf2"

// KeyGenerator uses PBKDF2 (Password-Based Key Derivation Function 2), part of PKCS #5 v2.0 (Password-Based
// Cryptography Specification).
//
// It can be used to derive a number of keys for various purposes from a given secret. This lets applications have a
// single secure secret, but avoid reusing that key in multiple incompatible contexts.
//
// The returned key is a binary. You may invoke functions in the `base64` module, such as
// `base64.StdEncoding.EncodeToString()`, to convert this binary into a textual representation.
//
// See http://tools.ietf.org/html/rfc2898#section-5.2
type KeyGenerator struct {
}

// Generate Returns a derived key suitable for use.
//
//   - `iterations` - defaults to 216,000 (2^16, OWASP recommended minimum for PBKDF2-SHA256);
//   - `length`     - a length in octets for the derived key. Defaults to 32;
//   - `digest`     - an hmac function to use as the pseudo-random function. Defaults to `sha256`;
//
// Security Note: The default of 216,000 iterations follows OWASP 2023 guidelines.
// For password hashing, consider using Argon2 instead for better security.
func (g *KeyGenerator) Generate(secret []byte, salt []byte, iterations int, length int, digest string) []byte {
	if iterations < 1 {
		iterations = 216000 // 2^16 - OWASP recommended minimum
	}
	if length < 1 {
		length = 32
	}
	sha2Func, _ := getSha2Func(digest)
	return pbkdf2.Key(secret, salt, iterations, length, sha2Func)
}
