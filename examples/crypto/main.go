package main

import (
	"encoding/base64"
	"github.com/syntax-framework/chain"
)

func main() {
	println("\nKeyGenerator")
	keyGenerator()

	println("\nMessageVerifier")
	messageVerifier()

	println("\nMessageEncryptor")
	messageEncryptor()
}

func keyGenerator() {
	secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	cookieSalt := []byte("encrypted cookie")
	signedCookieSalt := []byte("signed encrypted cookie")

	secret := chain.KeyGenerator.Generate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
	signSecret := chain.KeyGenerator.Generate(secretKeyBase, signedCookieSalt, 1000, 32, "sha256")

	println(base64.StdEncoding.EncodeToString(secret))
	println(base64.StdEncoding.EncodeToString(signSecret))
}

func messageVerifier() {
	message := []byte("This is content")
	secret := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	signed := chain.MessageVerifier.Sign(message, secret, "sha256")
	println(signed)

	verified, _ := chain.MessageVerifier.Verify([]byte(signed), secret)
	println(string(verified))
}

func messageEncryptor() {
	data := []byte("This is content")

	secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")
	cookieSalt := []byte("encrypted cookie")
	signedCookieSalt := []byte("signed encrypted cookie")
	secret := chain.KeyGenerator.Generate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
	signSecret := chain.KeyGenerator.Generate(secretKeyBase, signedCookieSalt, 1000, 32, "sha256")

	encrypted, _ := chain.MessageEncryptor.Encrypt(data, secret, signSecret)
	println(encrypted)

	decrypted, _ := chain.MessageEncryptor.Decrypt([]byte(encrypted), secret, signSecret)
	println(string(decrypted))
}
