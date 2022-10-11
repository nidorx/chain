package pubsub

import (
	"bytes"
	"github.com/syntax-framework/chain"
	"github.com/syntax-framework/chain/crypto"
)

var globalKeyring = chain.NewKeyring("chain.pubsub.keyring.salt", 1000, 32, "sha256")

var aad = append([]byte{byte(messageTypeEncrypt)}, []byte("chain.pubsub.aad")...)

// encryptPayload is used to encrypt a message before sending
func encryptPayload(keyring *crypto.Keyring, payload []byte) ([]byte, error) {
	encrypted, err := keyring.Encrypt(payload, aad)
	if err != nil {
		return nil, err
	}

	// return encrypted cipher text
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(messageTypeEncrypt))
	buf.Write(encrypted)
	return buf.Bytes(), nil
}

// decryptPayload is used to decrypt a message with a given keyring, and verify it's contents.
func decryptPayload(keyring *crypto.Keyring, encoded []byte) ([]byte, error) {
	return keyring.Decrypt(encoded[1:], aad)
}
