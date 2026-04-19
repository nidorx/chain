package crypto

import (
	"errors"
	"testing"
)

// --- SecureBytesCompare tests ---

func TestSecureBytesCompare_EqualSlices(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{0x01}},
		{"two bytes", []byte{0x01, 0x02}},
		{"ten bytes", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{"ascii string", []byte("hello world")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !SecureBytesCompare(tt.input, tt.input) {
				t.Errorf("SecureBytesCompare() = false, want true")
			}
		})
	}
}

func TestSecureBytesCompare_DifferentSlices(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		secret []byte
	}{
		{"single byte diff", []byte{0x01}, []byte{0x02}},
		{"two bytes diff", []byte{0x01, 0x02}, []byte{0x01, 0x03}},
		{"all bytes diff", []byte{0x00, 0x00, 0x00}, []byte{0xFF, 0xFF, 0xFF}},
		{"one byte diff", []byte{0x00, 0x00, 0x00}, []byte{0x00, 0xFF, 0x00}},
		{"ascii diff", []byte("hello"), []byte("world")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if SecureBytesCompare(tt.input, tt.secret) {
				t.Errorf("SecureBytesCompare() = true, want false")
			}
		})
	}
}

func TestSecureBytesCompare_DifferentLengths(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		secret []byte
	}{
		{"empty vs one", []byte{}, []byte{0x01}},
		{"one vs two", []byte{0x01}, []byte{0x01, 0x02}},
		{"short vs long", []byte{0xAB}, []byte{0xAB, 0xCD, 0xEF, 0x01}},
		{"10 vs 11", make([]byte, 10), make([]byte, 11)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if SecureBytesCompare(tt.input, tt.secret) {
				t.Errorf("SecureBytesCompare() = true, want false")
			}
		})
	}
}

func TestSecureBytesCompare_EmptySlices(t *testing.T) {
	if !SecureBytesCompare([]byte{}, []byte{}) {
		t.Errorf("SecureBytesCompare(empty, empty) = false, want true")
	}
}

func TestSecureBytesCompare_NilSlices(t *testing.T) {
	if !SecureBytesCompare(nil, nil) {
		t.Errorf("SecureBytesCompare(nil, nil) = false, want true")
	}
	// nil vs empty should be equal (both length 0)
	if !SecureBytesCompare(nil, []byte{}) {
		t.Errorf("SecureBytesCompare(nil, []) = false, want true")
	}
	if !SecureBytesCompare([]byte{}, nil) {
		t.Errorf("SecureBytesCompare([], nil) = false, want true")
	}
}

func TestSecureBytesCompare_SingleByte(t *testing.T) {
	if !SecureBytesCompare([]byte{0x00}, []byte{0x00}) {
		t.Errorf("SecureBytesCompare([0], [0]) = false, want true")
	}
	if !SecureBytesCompare([]byte{0xFF}, []byte{0xFF}) {
		t.Errorf("SecureBytesCompare([255], [255]) = false, want true")
	}
	if SecureBytesCompare([]byte{0x00}, []byte{0xFF}) {
		t.Errorf("SecureBytesCompare([0], [255]) = true, want false")
	}
}

func TestSecureBytesCompare_LargeSlices(t *testing.T) {
	large := make([]byte, 1000)
	for i := range large {
		large[i] = byte(i % 256)
	}

	if !SecureBytesCompare(large, large) {
		t.Errorf("SecureBytesCompare(large, large) = false, want true")
	}

	largeCopy := make([]byte, 1000)
	copy(largeCopy, large)
	if !SecureBytesCompare(large, largeCopy) {
		t.Errorf("SecureBytesCompare(large, largeCopy) = false, want true")
	}

	largeDiff := make([]byte, 1000)
	copy(largeDiff, large)
	largeDiff[999] = 0xFF
	if SecureBytesCompare(large, largeDiff) {
		t.Errorf("SecureBytesCompare(large, largeDiff) = true, want false")
	}
}

func TestSecureBytesCompare_Correctness(t *testing.T) {
	// Verify correctness across diverse inputs (not a timing test)
	tests := []struct {
		a, b   []byte
		expect bool
	}{
		{[]byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{[]byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{[]byte{1, 2, 3}, []byte{1, 3, 3}, false},
		{[]byte{1, 2, 3}, []byte{4, 2, 3}, false},
		{[]byte{0xFF}, []byte{0xFF}, true},
		{[]byte{0x00}, []byte{0x00}, true},
		{make([]byte, 100), make([]byte, 100), true},
		{[]byte("test"), []byte("test"), true},
		{[]byte("test"), []byte("Test"), false},
	}
	for _, tt := range tests {
		got := SecureBytesCompare(tt.a, tt.b)
		if got != tt.expect {
			t.Errorf("SecureBytesCompare(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expect)
		}
	}
}

func TestSecureBytesCompare_AllZeros(t *testing.T) {
	zeros1 := make([]byte, 64)
	zeros2 := make([]byte, 64)
	if !SecureBytesCompare(zeros1, zeros2) {
		t.Errorf("SecureBytesCompare(all zeros) = false, want true")
	}

	zerosDiff := make([]byte, 64)
	zerosDiff[32] = 0x01
	if SecureBytesCompare(zeros1, zerosDiff) {
		t.Errorf("SecureBytesCompare(zeros vs one-diff) = true, want false")
	}
}

func TestSecureBytesCompare_AllOnes(t *testing.T) {
	ones1 := make([]byte, 64)
	for i := range ones1 {
		ones1[i] = 0xFF
	}
	ones2 := make([]byte, 64)
	for i := range ones2 {
		ones2[i] = 0xFF
	}
	if !SecureBytesCompare(ones1, ones2) {
		t.Errorf("SecureBytesCompare(all ones) = false, want true")
	}

	onesDiff := make([]byte, 64)
	for i := range onesDiff {
		onesDiff[i] = 0xFF
	}
	onesDiff[0] = 0xFE
	if SecureBytesCompare(ones1, onesDiff) {
		t.Errorf("SecureBytesCompare(ones vs one-diff) = true, want false")
	}
}

// --- ValidateKey tests ---

func TestValidateKeyUtils_ValidSizes(t *testing.T) {
	validSizes := []int{16, 24, 32}
	for _, size := range validSizes {
		t.Run("size_"+string(rune('0'+size/10))+string(rune('0'+size%10)), func(t *testing.T) {
			key := make([]byte, size)
			err := ValidateKey(key)
			if err != nil {
				t.Errorf("ValidateKey(%d bytes) returned error: %v, want nil", size, err)
			}
		})
	}
}

func TestValidateKeyUtils_InvalidSizes(t *testing.T) {
	invalidSizes := []int{0, 1, 15, 17, 23, 25, 31, 33, 64}
	for _, size := range invalidSizes {
		t.Run("size_"+string(rune('0'+size/10))+string(rune('0'+size%10)), func(t *testing.T) {
			key := make([]byte, size)
			err := ValidateKey(key)
			if err == nil {
				t.Errorf("ValidateKey(%d bytes) returned nil, want error", size)
			}
		})
	}
}

func TestValidateKey_EmptyKey(t *testing.T) {
	err := ValidateKey([]byte{})
	if err == nil {
		t.Errorf("ValidateKey(empty) returned nil, want error")
	}
	if !errors.Is(err, ErrKeySize) {
		t.Errorf("ValidateKey(empty) error = %v, want ErrKeySize", err)
	}
}

func TestValidateKey_NilKey(t *testing.T) {
	err := ValidateKey(nil)
	if err == nil {
		t.Errorf("ValidateKey(nil) returned nil, want error")
	}
	if !errors.Is(err, ErrKeySize) {
		t.Errorf("ValidateKey(nil) error = %v, want ErrKeySize", err)
	}
}

func TestValidateKey_SingleByte(t *testing.T) {
	err := ValidateKey([]byte{0x01})
	if err == nil {
		t.Errorf("ValidateKey(1 byte) returned nil, want error")
	}
	if !errors.Is(err, ErrKeySize) {
		t.Errorf("ValidateKey(1 byte) error = %v, want ErrKeySize", err)
	}
}

// --- Error constant tests ---

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{"ErrKeySize", ErrKeySize, "key size must be 16, 24 or 32 bytes"},
		{"ErrInvalidSignature", ErrInvalidSignature, "invalid signature"},
		{"ErrInvalidMessage", ErrInvalidMessage, "invalid message"},
		{"ErrKeyringEmpty", ErrKeyringEmpty, "no installed keys"},
		{"ErrKeyringCannotDecrypt", ErrKeyringCannotDecrypt, "no installed keys could decrypt the message"},
		{"ErrKeyringCannotVerify", ErrKeyringCannotVerify, "no installed keys could verify the message"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}
			if tt.err.Error() != tt.message {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.message)
			}
		})
	}
}

// --- Mapping tests ---

func TestHmacSha2ToAlgoName_Completeness(t *testing.T) {
	expected := map[string][]byte{
		"sha256": []byte("HS256"),
		"sha384": []byte("HS384"),
		"sha512": []byte("HS512"),
	}
	if len(hmacSha2ToAlgoName) != len(expected) {
		t.Errorf("hmacSha2ToAlgoName has %d entries, want %d", len(hmacSha2ToAlgoName), len(expected))
	}
	for k, v := range expected {
		got, ok := hmacSha2ToAlgoName[k]
		if !ok {
			t.Errorf("hmacSha2ToAlgoName missing key %q", k)
		}
		if !SecureBytesCompare(got, v) {
			t.Errorf("hmacSha2ToAlgoName[%q] = %v, want %v", k, got, v)
		}
	}
}

func TestHmacSha2ToDigestType_Completeness(t *testing.T) {
	expected := map[string]string{
		"HS256": "sha256",
		"HS384": "sha384",
		"HS512": "sha512",
	}
	if len(hmacSha2ToDigestType) != len(expected) {
		t.Errorf("hmacSha2ToDigestType has %d entries, want %d", len(hmacSha2ToDigestType), len(expected))
	}
	for k, v := range expected {
		got, ok := hmacSha2ToDigestType[k]
		if !ok {
			t.Errorf("hmacSha2ToDigestType missing key %q", k)
		}
		if got != v {
			t.Errorf("hmacSha2ToDigestType[%q] = %q, want %q", k, got, v)
		}
	}
}

func TestMappingRoundtrip(t *testing.T) {
	// digest -> algo -> digest
	digests := []string{"sha256", "sha384", "sha512"}
	for _, digest := range digests {
		algoName, ok := hmacSha2ToAlgoName[digest]
		if !ok {
			t.Errorf("hmacSha2ToAlgoName missing digest %q", digest)
			continue
		}
		algoStr := string(algoName)
		roundtrip, ok := hmacSha2ToDigestType[algoStr]
		if !ok {
			t.Errorf("hmacSha2ToDigestType missing algo %q", algoStr)
			continue
		}
		if roundtrip != digest {
			t.Errorf("roundtrip %q -> %q -> %q, want %q", digest, algoStr, roundtrip, digest)
		}
	}

	// algo -> digest -> algo
	algos := []string{"HS256", "HS384", "HS512"}
	for _, algo := range algos {
		digest, ok := hmacSha2ToDigestType[algo]
		if !ok {
			t.Errorf("hmacSha2ToDigestType missing algo %q", algo)
			continue
		}
		roundtripBytes, ok := hmacSha2ToAlgoName[digest]
		if !ok {
			t.Errorf("hmacSha2ToAlgoName missing digest %q", digest)
			continue
		}
		roundtrip := string(roundtripBytes)
		if roundtrip != algo {
			t.Errorf("roundtrip %q -> %q -> %q, want %q", algo, digest, roundtrip, algo)
		}
	}
}

func TestMappingInverseConsistency(t *testing.T) {
	// Verify that for every digest, the reverse mapping is consistent
	for digest, algoBytes := range hmacSha2ToAlgoName {
		algo := string(algoBytes)
		reverseDigest, ok := hmacSha2ToDigestType[algo]
		if !ok {
			t.Errorf("no reverse mapping for algo %q", algo)
			continue
		}
		if reverseDigest != digest {
			t.Errorf("inconsistent mapping: %q -> %q -> %q, want %q", digest, algo, reverseDigest, digest)
		}
	}
}
