package chain

import (
	"log/slog"

	"github.com/nidorx/chain/crypto"
)

// NewKeyring starts a Keyring that will be updated whenever SecretKeySync() is invoked
//
//   - `salt`			- a salt used with SecretKeyBase to generate a secret
//   - `iterations` 	- defaults to 216,000 (2^16, OWASP recommended minimum for PBKDF2-SHA256)
//   - `length`     	- a length in octets for the derived key. Defaults to 32
//   - `digest`     	- a hmac function to use as the pseudo-random function. Defaults to `sha256`
//
// Security Note: The default of 216,000 iterations follows OWASP 2023 guidelines.
// For password hashing, consider using Argon2 instead for better security.
func NewKeyring(salt string, iterations int, length int, digest string) *crypto.Keyring {

	if iterations < 1 {
		iterations = 216000 // 2^16 - OWASP recommended minimum
	}
	if length < 1 {
		length = 32
	}
	if digest == "" {
		digest = "sha256"
	}
	k := &crypto.Keyring{}

	SecretKeySync(func(secretKeyBase string) {
		key := crypt.KeyGenerate([]byte(secretKeyBase), []byte(salt), iterations, length, digest)
		if err := k.AddKey(key); err != nil {
			slog.Error("[chain.keyring] error deriving key from SecretKeyBase", slog.Any("error", err))
			return
		}
	})

	return k
}
