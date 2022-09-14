package crypto

import (
	"bytes"
	"math/rand"
	"testing"
)

func Test_KeyGenerator(t *testing.T) {
	generator := KeyGenerator{}

	secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	salt := make([]byte, 16)
	rand.Read(salt)

	signingSaltA := generator.Generate(secretKeyBase, salt, 0, 0, "")
	signingSaltB := generator.Generate(secretKeyBase, salt, 0, 0, "")

	if !bytes.Equal(signingSaltA, signingSaltB) {
		t.Errorf("KeyGenerator.Generate() failed: Invalid Result\n actual: %v\n expected: %v", string(signingSaltB), string(signingSaltA))
	}

	signingSaltB = generator.Generate(secretKeyBase, salt, 5, 0, "")
	if bytes.Equal(signingSaltA, signingSaltB) {
		t.Errorf("KeyGenerator.Generate() failed: Invalid Result\n actual: %v\n expected: any other value", string(signingSaltB))
	}

	signingSaltB = generator.Generate(secretKeyBase, salt, 0, 16, "")
	if bytes.Equal(signingSaltA, signingSaltB) {
		t.Errorf("KeyGenerator.Generate() failed: Invalid Result\n actual: %v\n expected: any other value", string(signingSaltB))
	}

	signingSaltB = generator.Generate(secretKeyBase, salt, 0, 0, "sha384")
	if bytes.Equal(signingSaltA, signingSaltB) {
		t.Errorf("KeyGenerator.Generate() failed: Invalid Result\n actual: %v\n expected: any other value", string(signingSaltB))
	}

	signingSaltB = generator.Generate(secretKeyBase, salt, 0, 0, "sha512")
	if bytes.Equal(signingSaltA, signingSaltB) {
		t.Errorf("KeyGenerator.Generate() failed: Invalid Result\n actual: %v\n expected: any other value", string(signingSaltB))
	}
}
