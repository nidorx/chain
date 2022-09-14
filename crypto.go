package chain

import "github.com/syntax-framework/chain/crypto"

type chainCrypto struct {
	Generator crypto.KeyGenerator
	Encryptor crypto.MessageEncryptor
	Verifier  crypto.MessageVerifier
}

// Public instances
var (
	KeyGenerator     = crypto.KeyGenerator{}
	MessageEncryptor = crypto.MessageEncryptor{}
	MessageVerifier  = crypto.MessageVerifier{}
	Crypto           = chainCrypto{}
)
