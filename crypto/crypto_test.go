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

func Test_MessageVerifier(t *testing.T) {
	verifier := MessageVerifier{}

	message := []byte("! \" # $ % & ' ( ) * + , - . / 0 1 2 3 4 5 6 7 8 9 : ; < = > ?Ā ā Ă ă Ą ą Ć ć Ĉ ĉ Ċ ċ Č")
	secret := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	digests := []string{"sha512", "sha384", "sha256", "invalid"}
	for _, digest := range digests {
		t.Run(digest, func(t *testing.T) {
			signed := verifier.Sign(secret, message, digest)
			verified, err := verifier.Verify(secret, []byte(signed))
			if err != nil {
				t.Errorf("MessageVerifier failed:\n   error: %v", err)
			}

			if !bytes.Equal(message, verified) {
				t.Errorf("MessageVerifier failed: Invalid Result\n   digest: %v\n actual: %v\n expected: %v", digest, string(verified), string(message))
			}

		})
	}
}

func Test_MessageEncryptor(t *testing.T) {
	encryptor := MessageEncryptor{}
	generator := KeyGenerator{}

	//	secret_key_base = "072d1e0157c008193fe48a670cce031faa4e..."
	//	encrypted_cookie_salt = "encrypted cookie"
	//	encrypted_signed_cookie_salt = "signed encrypted cookie"
	//
	//	secret = KeyGenerator.generate(secret_key_base, encrypted_cookie_salt)
	//	sign_secret = KeyGenerator.generate(secret_key_base, encrypted_signed_cookie_salt)
	//
	//	data = "José"
	//	encrypted = MessageEncryptor.encrypt(data, secret, sign_secret)
	//	MessageEncryptor.decrypt(encrypted, secret, sign_secret)
	//	    expects "José"

	secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")
	cookieSalt := []byte("encrypted cookie")
	signedCookieSalt := []byte("signed encrypted cookie")

	secret := generator.Generate(secretKeyBase, cookieSalt, 0, 0, "")
	aad := generator.Generate(secretKeyBase, signedCookieSalt, 0, 0, "")

	message := []byte("José")

	encrypted, err := encryptor.Encrypt(secret, message, aad)
	if err != nil {
		t.Errorf("MessageEncryptor.Encrypt() failed:\n   error: %v", err)
	}
	decrypted, err := encryptor.Decrypt(secret, []byte(encrypted), aad)
	if err != nil {
		t.Errorf("MessageEncryptor.Decrypt() failed:\n   error: %v", err)
	}
	if !bytes.Equal(message, decrypted) {
		t.Errorf("MessageEncryptor failed: Invalid Result\n actual: %v\n expected: %v", string(decrypted), string(message))
	}

}
