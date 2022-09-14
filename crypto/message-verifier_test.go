package crypto

import (
	"bytes"
	"testing"
)

func Test_MessageVerifier(t *testing.T) {
	verifier := MessageVerifier{}

	message := []byte("! \" # $ % & ' ( ) * + , - . / 0 1 2 3 4 5 6 7 8 9 : ; < = > ?Ā ā Ă ă Ą ą Ć ć Ĉ ĉ Ċ ċ Č")
	secret := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	digests := []string{"sha512", "sha384", "sha256", "invalid"}
	for _, digest := range digests {
		t.Run(digest, func(t *testing.T) {
			signed := verifier.Sign(message, secret, digest)
			verified, err := verifier.Verify([]byte(signed), secret)
			if err != nil {
				t.Errorf("MessageVerifier failed:\n   error: %v", err)
			}

			if !bytes.Equal(message, verified) {
				t.Errorf("MessageVerifier failed: Invalid Result\n   digest: %v\n actual: %v\n expected: %v", digest, string(verified), string(message))
			}

		})
	}
}
