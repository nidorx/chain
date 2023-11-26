package chain

import (
	"bytes"
	"sync"

	"github.com/nidorx/chain/crypto"
)

type SecretKeySyncFunc func(key string)

var (
	// secretKeys stores the key data received by SetSecretKeyBase() function. It is ordered in such a way where the last
	// key (index len(secretKeys)-1) is the primary key (the most recent key for rotation)
	secretKeys         [][]byte
	secretKeysMutex    = sync.RWMutex{}
	secretKeySync      []SecretKeySyncFunc
	secretKeySyncMutex = sync.RWMutex{}
)

// SetSecretKeyBase see SecretKeyBase()
func SetSecretKeyBase(secret string) error {
	key := []byte(secret)
	if err := crypto.ValidateKey(key); err != nil {
		return err
	}

	secretKeysMutex.Lock()
	defer secretKeysMutex.Unlock()

	if l := len(secretKeys); l > 0 && bytes.Equal(secretKeys[l-1], key) {
		return nil
	}
	secretKeys = append(secretKeys, key)

	secretKeySyncMutex.RLock()
	defer secretKeySyncMutex.RUnlock()
	for _, sync := range secretKeySync {
		sync(secret)
	}
	return nil
}

// SecretKeyBase A secret key used to verify and encrypt data.
//
// This data must be never used directly, always use chain.Crypto().KeyGenerate() to derive keys from it
func SecretKeyBase() string {
	secretKeysMutex.RLock()
	defer secretKeysMutex.RUnlock()
	if l := len(secretKeys); l > 0 {
		return string(secretKeys[l-1])
	}
	return ""
}

// SecretKeys gets the list of all SecretKeyBase that have been defined. Can be used in key rotation algorithms
//
// The LAST item in the list is the most recent key (primary key)
func SecretKeys() []string {
	secretKeysMutex.RLock()
	defer secretKeysMutex.RUnlock()
	keys := make([]string, len(secretKeys))
	l := len(keys) - 1
	for i, key := range keys {
		keys[l-i] = string(key)
	}
	return keys
}

// SecretKeySync is used to transmit SecretKeyBase changes
func SecretKeySync(sync SecretKeySyncFunc) (cancel func()) {

	cancel = func() {
		secretKeySyncMutex.Lock()
		var syncs []SecretKeySyncFunc
		for _, s := range secretKeySync {
			if &s != &sync {
				syncs = append(syncs, s)
			}
		}
		secretKeySync = syncs
		secretKeySyncMutex.Unlock()
	}

	secretKeySyncMutex.Lock()
	secretKeySync = append(secretKeySync, sync)
	secretKeySyncMutex.Unlock()

	for _, key := range SecretKeys() {
		sync(key)
	}

	return cancel
}
