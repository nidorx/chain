package chain

import (
	"github.com/nidorx/chain/crypto"
	"github.com/rs/zerolog/log"
)

// NewKeyring starts a Keyring that will be updated whenever SecretKeySync() is invoked
//
//   - `salt`			- a salt used with SecretKeyBase to generate a secret
//   - `iterations` 	- defaults to 1000 (increase to at least 2^16 if used for passwords)
//   - `length`     	- a length in octets for the derived key. Defaults to 32
//   - `digest`     	- a hmac function to use as the pseudo-random function. Defaults to `sha256`
func NewKeyring(salt string, iterations int, length int, digest string) *crypto.Keyring {

	if iterations < 1 {
		iterations = 1000
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
			log.Error().Err(err).Msg(_l("[chain.keyring] error deriving key from SecretKeyBase"))
			return
		}
	})

	return k
}
