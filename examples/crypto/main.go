package main

import (
	"encoding/base64"
	"github.com/nidorx/chain"
)

func main() {
	println("\nKeyGenerator")
	keyGenerator()

	println("\nMessageVerifier")
	messageVerifier()

	println("\nMessageEncryptor")
	messageEncryptor()

	println("\nKeyring")
	keyring()

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

	encryptionKey := chain.Crypto().KeyGenerate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
	aad := []byte("purpose: database key")

	encrypted, _ := chain.Crypto().MessageEncrypt(encryptionKey, data, aad)
	println(encrypted)

	decrypted, _ := chain.Crypto().MessageDecrypt(encryptionKey, []byte(encrypted), aad)
	println(string(decrypted))
}

func keyring() {
	aad := []byte("purpose: database key")

	var myKeyring = chain.NewKeyring("SALT", 1000, 32, "sha256")

	// moment 1, set global key
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}

	encryptedA, _ := myKeyring.Encrypt([]byte("Jack"), aad)
	println(base64.StdEncoding.EncodeToString(encryptedA))

	// moment 2, update global key
	if err := chain.SetSecretKeyBase("fe6d1fed11fa60277fb6a2f73efb8be2"); err != nil {
		panic(err)
	}

	// encrypt using new key
	encryptedB, _ := myKeyring.Encrypt([]byte("Jack"), aad)
	println(base64.StdEncoding.EncodeToString(encryptedB))

	// decrypt value encrypted by old key
	decryptedA, _ := myKeyring.Decrypt(encryptedA, aad)
	println(string(decryptedA))

	// decrypt value encrypted by new key
	decryptedB, _ := myKeyring.Decrypt(encryptedB, aad)
	println(string(decryptedB))
}
