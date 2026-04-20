package pubsub

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/crypto"
)

var globalKeyring = func() *crypto.Keyring {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	return chain.NewKeyring(
		hex.EncodeToString(salt),
		216000,
		32,
		"sha256",
	)
}()

var aad = append([]byte{byte(MessageTypeEncrypt)}, []byte("chain.pubsub.aad")...)

// encryptPayload is used to encrypt a message before sending
func encryptPayload(keyring *crypto.Keyring, payload []byte) ([]byte, error) {
	encrypted, err := keyring.Encrypt(payload, aad)
	if err != nil {
		return nil, err
	}

	// return encrypted cipher text
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(MessageTypeEncrypt))
	buf.Write(encrypted)
	return buf.Bytes(), nil
}

// decryptPayload is used to decrypt a message with a given keyring, and verify it's contents.
func decryptPayload(keyring *crypto.Keyring, encoded []byte) ([]byte, error) {
	return keyring.Decrypt(encoded[1:], aad)
}
