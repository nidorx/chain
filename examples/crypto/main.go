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

	secret := chain.Crypto().KeyGenerate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
	signSecret := chain.Crypto().KeyGenerate(secretKeyBase, signedCookieSalt, 1000, 32, "sha256")

	println(base64.StdEncoding.EncodeToString(secret))
	println(base64.StdEncoding.EncodeToString(signSecret))
}

func messageVerifier() {
	message := []byte("This is content")
	secret := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	signed := chain.Crypto().MessageSign(secret, message, "sha256")
	println(signed)

	verified, _ := chain.Crypto().MessageVerify(secret, []byte(signed))
	println(string(verified))
}

func messageEncryptor() {
	data := []byte("This is content")

	secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

	cookieSalt := []byte("encrypted cookie")
	signedCookieSalt := []byte("signed encrypted cookie")

	encryptionKey := chain.Crypto().KeyGenerate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
	aad := chain.Crypto().KeyGenerate(secretKeyBase, signedCookieSalt, 1000, 32, "sha256")

	encrypted, _ := chain.Crypto().MessageEncrypt(encryptionKey, data, aad)
	println(encrypted)

	decrypted, _ := chain.Crypto().MessageDecrypt(encryptionKey, []byte(encrypted), aad)
	println(string(decrypted))
}
