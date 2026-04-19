package crypto

import (
	"bytes"
	"sync"
	"testing"
)

func TestKeyGenerator_Generate_BasicDefaults(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")

	key := g.Generate(secret, salt, 0, 0, "")

	if len(key) != 32 {
		t.Errorf("expected key length 32, got %d", len(key))
	}
	if len(key) == 0 {
		t.Error("expected non-empty key")
	}
}

func TestKeyGenerator_Generate_ExplicitIterations(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	digest := "sha256"
	length := 32

	tests := []struct {
		name       string
		iterations int
	}{
		{"1000 iterations", 1000},
		{"10000 iterations", 10000},
		{"216000 iterations", 216000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := g.Generate(secret, salt, tt.iterations, length, digest)
			if len(key) != length {
				t.Errorf("expected key length %d, got %d", length, len(key))
			}
		})
	}
}

func TestKeyGenerator_Generate_DifferentLengths(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	digest := "sha256"

	tests := []struct {
		name   string
		length int
	}{
		{"16 bytes (AES-128)", 16},
		{"24 bytes (AES-192)", 24},
		{"32 bytes (AES-256)", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := g.Generate(secret, salt, iterations, tt.length, digest)
			if len(key) != tt.length {
				t.Errorf("expected key length %d, got %d", tt.length, len(key))
			}
		})
	}
}

func TestKeyGenerator_Generate_SHA256Digest(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32

	key := g.Generate(secret, salt, iterations, length, "sha256")

	if len(key) != length {
		t.Errorf("expected key length %d, got %d", length, len(key))
	}
}

func TestKeyGenerator_Generate_SHA384Digest(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32

	key := g.Generate(secret, salt, iterations, length, "sha384")

	if len(key) != length {
		t.Errorf("expected key length %d, got %d", length, len(key))
	}
}

func TestKeyGenerator_Generate_SHA512Digest(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32

	key := g.Generate(secret, salt, iterations, length, "sha512")

	if len(key) != length {
		t.Errorf("expected key length %d, got %d", length, len(key))
	}
}

func TestKeyGenerator_Generate_InvalidDigestDefaultsToSHA256(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32

	keyInvalid := g.Generate(secret, salt, iterations, length, "invalid-digest")
	keySHA256 := g.Generate(secret, salt, iterations, length, "sha256")

	if !bytes.Equal(keyInvalid, keySHA256) {
		t.Error("invalid digest should default to sha256")
	}
}

func TestKeyGenerator_Generate_Deterministic(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32
	digest := "sha256"

	key1 := g.Generate(secret, salt, iterations, length, digest)
	key2 := g.Generate(secret, salt, iterations, length, digest)

	if !bytes.Equal(key1, key2) {
		t.Error("same inputs should produce identical keys")
	}
}

func TestKeyGenerator_Generate_DifferentSaltProducesDifferentKey(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt1 := []byte("salt-one")
	salt2 := []byte("salt-two")
	iterations := 1000
	length := 32
	digest := "sha256"

	key1 := g.Generate(secret, salt1, iterations, length, digest)
	key2 := g.Generate(secret, salt2, iterations, length, digest)

	if bytes.Equal(key1, key2) {
		t.Error("different salts should produce different keys")
	}
}

func TestKeyGenerator_Generate_DifferentSecretProducesDifferentKey(t *testing.T) {
	g := &KeyGenerator{}
	secret1 := []byte("secret-one")
	secret2 := []byte("secret-two")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32
	digest := "sha256"

	key1 := g.Generate(secret1, salt, iterations, length, digest)
	key2 := g.Generate(secret2, salt, iterations, length, digest)

	if bytes.Equal(key1, key2) {
		t.Error("different secrets should produce different keys")
	}
}

func TestKeyGenerator_Generate_DifferentIterationsProduceDifferentKeys(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	length := 32
	digest := "sha256"

	key1 := g.Generate(secret, salt, 1000, length, digest)
	key2 := g.Generate(secret, salt, 10000, length, digest)

	if bytes.Equal(key1, key2) {
		t.Error("different iteration counts should produce different keys")
	}
}

func TestKeyGenerator_Generate_EmptySecretNonEmptySalt(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("")
	salt := []byte("my-salt")
	iterations := 1000
	length := 32
	digest := "sha256"

	key := g.Generate(secret, salt, iterations, length, digest)

	if len(key) != length {
		t.Errorf("expected key length %d, got %d", length, len(key))
	}
}

func TestKeyGenerator_Generate_EmptySaltNonEmptySecret(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("")
	iterations := 1000
	length := 32
	digest := "sha256"

	key := g.Generate(secret, salt, iterations, length, digest)

	if len(key) != length {
		t.Errorf("expected key length %d, got %d", length, len(key))
	}
}

func TestKeyGenerator_Generate_ZeroIterationsDefaults(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	length := 32
	digest := "sha256"

	keyZero := g.Generate(secret, salt, 0, length, digest)
	keyDefault := g.Generate(secret, salt, 216000, length, digest)

	if !bytes.Equal(keyZero, keyDefault) {
		t.Error("zero iterations should default to 216000")
	}
}

func TestKeyGenerator_Generate_NegativeIterationsDefaults(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	length := 32
	digest := "sha256"

	keyNeg := g.Generate(secret, salt, -1, length, digest)
	keyDefault := g.Generate(secret, salt, 216000, length, digest)

	if !bytes.Equal(keyNeg, keyDefault) {
		t.Error("negative iterations should default to 216000")
	}
}

func TestKeyGenerator_Generate_ZeroLengthDefaults(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	digest := "sha256"

	key := g.Generate(secret, salt, iterations, 0, digest)

	if len(key) != 32 {
		t.Errorf("zero length should default to 32, got %d", len(key))
	}
}

func TestKeyGenerator_Generate_KeyLengthVerification(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	iterations := 1000
	digest := "sha256"

	lengths := []int{16, 24, 32, 48, 64, 128}

	for _, length := range lengths {
		t.Run("length-"+string(rune(length)), func(t *testing.T) {
			key := g.Generate(secret, salt, iterations, length, digest)
			if len(key) != length {
				t.Errorf("expected key length %d, got %d", length, len(key))
			}
		})
	}
}

func BenchmarkKeyGenerator_Generate(b *testing.B) {
	g := &KeyGenerator{}
	secret := []byte("benchmark-secret")
	salt := []byte("benchmark-salt")
	length := 32
	digest := "sha256"

	benchmarks := []struct {
		name       string
		iterations int
	}{
		{"1000 iterations", 1000},
		{"10000 iterations", 10000},
		{"216000 iterations", 216000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				g.Generate(secret, salt, bm.iterations, length, digest)
			}
		})
	}
}

func TestKeyGenerator_Generate_KeyRotation(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("rotation-secret")
	baseSalt := []byte("app-salt")
	iterations := 1000
	length := 32
	digest := "sha256"

	// Generate multiple keys from the same secret using different salt versions
	keys := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		salt := append([]byte{}, baseSalt...)
		salt = append(salt, byte(i))
		keys[i] = g.Generate(secret, salt, iterations, length, digest)
	}

	// All keys should be different
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			if bytes.Equal(keys[i], keys[j]) {
				t.Errorf("key rotation %d and %d produced identical keys", i, j)
			}
		}
	}

	// All keys should have correct length
	for i, key := range keys {
		if len(key) != length {
			t.Errorf("rotation key %d has wrong length: expected %d, got %d", i, length, len(key))
		}
	}
}

func TestKeyGenerator_Generate_Concurrent(t *testing.T) {
	g := &KeyGenerator{}
	secret := []byte("concurrent-secret")
	salt := []byte("concurrent-salt")
	iterations := 1000
	length := 32
	digest := "sha256"

	const goroutines = 100
	const iterationsPerGoroutine = 10

	var wg sync.WaitGroup
	results := make([][]byte, goroutines*iterationsPerGoroutine)
	var mu sync.Mutex

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				key := g.Generate(secret, salt, iterations, length, digest)
				mu.Lock()
				results[i*iterationsPerGoroutine+j] = key
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	// Verify all keys have correct length
	for i, key := range results {
		if len(key) != length {
			t.Errorf("concurrent key %d has wrong length: expected %d, got %d", i, length, len(key))
		}
	}

	// Verify deterministic: all keys should be identical
	firstKey := results[0]
	for i, key := range results {
		if !bytes.Equal(key, firstKey) {
			t.Errorf("concurrent key %d differs from first key", i)
		}
	}
}
