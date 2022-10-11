package chain

import "github.com/syntax-framework/chain/crypto"

var (
	crypt        = &cryptoShortcuts{}
	msgEncryptor = crypto.MessageEncryptor{}
	msgVerifier  = crypto.MessageVerifier{}
	keyGenerator = crypto.KeyGenerator{}
)

// Crypto get the reference to a structure that has shortcut to all encryption related functions
func Crypto() *cryptoShortcuts {
	return crypt
}

type cryptoShortcuts struct{}

// Encrypt is used to encrypt a data with a given key.
func (c *cryptoShortcuts) Encrypt(secret []byte, data []byte, aad []byte) (encrypted []byte, err error) {
	return crypto.Encrypt(secret, data, aad)
}

// Decrypt is used to decrypt a message with a given key, and verify it's contents.
func (c *cryptoShortcuts) Decrypt(secret []byte, encrypted []byte, aad []byte) (plain []byte, err error) {
	return crypto.Decrypt(secret, encrypted, aad)
}

// KeyGenerate Returns a derived key suitable for use.
//
// See crypto.KeyGenerator.Generate()
func (c *cryptoShortcuts) KeyGenerate(secret []byte, salt []byte, iterations int, length int, digest string) []byte {
	return keyGenerator.Generate(secret, salt, iterations, length, digest)
}

// MessageSign generates a signed message for the provided value.
//
// See crypto.MessageVerifier.Sign()
func (c *cryptoShortcuts) MessageSign(secret []byte, message []byte, digest string) string {
	return msgVerifier.Sign(secret, message, digest)
}

// MessageVerify decodes and verifies the encoded binary was not tampered with.
//
// See crypto.MessageVerifier.Verify()
func (c *cryptoShortcuts) MessageVerify(secret []byte, signed []byte) (decoded []byte, err error) {
	return msgVerifier.Verify(secret, signed)
}

// MessageEncrypt encrypts and authenticates a message using AES128-GCM mode.
//
// See crypto.MessageEncryptor.Encrypt()
func (c *cryptoShortcuts) MessageEncrypt(secret []byte, content []byte, aad []byte) (encoded string, err error) {
	return msgEncryptor.Encrypt(secret, content, aad)
}

// MessageDecrypt decrypt a message using authenticated encryption.
//
// See crypto.MessageEncryptor.Decrypt()
func (c *cryptoShortcuts) MessageDecrypt(secret []byte, encoded []byte, aad []byte) (content []byte, err error) {
	return msgEncryptor.Decrypt(secret, encoded, aad)
}
