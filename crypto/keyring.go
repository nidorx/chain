package crypto

import (
	"bytes"
	"errors"
	"sync"
)

var (
	msgVerifier             = MessageVerifier{}
	msgEncryptor            = MessageEncryptor{}
	ErrKeyringEmpty         = errors.New("no installed keys")
	ErrKeyringCannotDecrypt = errors.New("no installed keys could decrypt the message")
	ErrKeyringCannotVerify  = errors.New("no installed keys could verify the message")
)

type Keyring struct {
	// Keys stores the key data used during encryption and decryption. It is ordered in such a way where the first key
	// (index 0) is the primary key, which is used for encrypting messages, and is the first key tried during
	// message decryption.
	keys [][]byte

	// The keyring lock is used while performing IO operations on the keyring.
	mutex sync.RWMutex
}

// AddKey will install a new key on the ring. Adding a key to the ring will make it available for use in decryption. If
// the key already exists on the ring, this function will just return noop.
func (k *Keyring) AddKey(key []byte) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if k.keys == nil {
		k.keys = make([][]byte, 0)
	}

	// No-op if key is already installed
	for _, installedKey := range k.keys {
		if bytes.Equal(installedKey, key) {
			return nil
		}
	}

	var primaryKey []byte
	if len(k.keys) > 0 {
		primaryKey = k.keys[0]
	}
	if primaryKey == nil {
		primaryKey = key
	}
	newKeys := [][]byte{primaryKey}
	for _, it := range k.keys {
		if !bytes.Equal(it, primaryKey) {
			newKeys = append(newKeys, it)
		}
	}
	k.keys = newKeys
	return nil
}

// GetKeys returns the current set of keys on the ring.
func (k *Keyring) GetKeys() [][]byte {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	return k.keys
}

// GetPrimaryKey returns the key on the ring at position 0. This is the key used
// for encrypting messages, and is the first key tried for decrypting messages.
func (k *Keyring) GetPrimaryKey() (key []byte) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	if len(k.keys) > 0 {
		key = k.keys[0]
	}
	return
}

// Encrypt is used to encrypt a data using Keyring primary key.
func (k *Keyring) Encrypt(data, aad []byte) (cipherText []byte, err error) {
	key := k.GetPrimaryKey()
	if key == nil {
		return nil, ErrKeyringEmpty
	}

	// return encrypted cipher text
	return Encrypt(key, data, aad)
}

// Decrypt is used to decrypt a message using Keyring keys, and verify it's contents.
func (k *Keyring) Decrypt(cipherText, aad []byte) (plain []byte, err error) {
	keys := k.GetKeys()
	for _, key := range keys {
		plain, err = Decrypt(key, cipherText, aad)
		if err == nil {
			return
		}
	}

	return nil, ErrKeyringCannotDecrypt
}

// MessageEncrypt a message using authenticated encryption.
func (k *Keyring) MessageEncrypt(content []byte, aad []byte) (encrypted string, err error) {
	key := k.GetPrimaryKey()
	if key == nil {
		return "", ErrKeyringEmpty
	}
	return msgEncryptor.Encrypt(key, content, aad)
}

// MessageDecrypt a message using authenticated encryption.
func (k *Keyring) MessageDecrypt(encrypted []byte, aad []byte) ([]byte, error) {
	keys := k.GetKeys()
	for _, key := range keys {
		message, err := msgEncryptor.Decrypt(key, encrypted, aad)
		if err == nil {
			return message, nil
		}
	}

	return nil, ErrKeyringCannotDecrypt
}

// MessageSign Generates a signed message for the provided value.
func (k *Keyring) MessageSign(message []byte, digest string) (string, error) {
	key := k.GetPrimaryKey()
	if key == nil {
		return "", ErrKeyringEmpty
	}

	return msgVerifier.Sign(key, message, digest), nil
}

// MessageVerify Decodes and verifies the encoded binary was not tampered with.
func (k *Keyring) MessageVerify(signed []byte) (decoded []byte, err error) {
	keys := k.GetKeys()
	for _, key := range keys {
		decoded, err = msgVerifier.Verify(key, signed)
		if err == nil {
			return
		}
	}
	return nil, ErrKeyringCannotVerify
}
