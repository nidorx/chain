package crypto

import (
	"bytes"
	"testing"
)

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
	signSecret := generator.Generate(secretKeyBase, signedCookieSalt, 0, 0, "")

	message := []byte("José")

	encrypted, err := encryptor.Encrypt(message, secret, signSecret)
	println(encrypted)
	if err != nil {
		t.Errorf("MessageEncryptor.Encrypt() failed:\n   error: %v", err)
	}
	decrypted, err := encryptor.Decrypt([]byte(encrypted), secret, signSecret)
	if err != nil {
		t.Errorf("MessageEncryptor.Decrypt() failed:\n   error: %v", err)
	}
	if !bytes.Equal(message, decrypted) {
		t.Errorf("MessageEncryptor failed: Invalid Result\n actual: %v\n expected: %v", string(decrypted), string(message))
	}

}
