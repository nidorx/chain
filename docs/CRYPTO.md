# Crypto

## SecretKeyBase

In order to facilitate the maintenance of keys in your project, Chain provides a mechanism for managing a global key.

The global key should not be used directly, it is recommended to use `Keyring` or `KeyGenerator` to derive keys
from `chain.SecretKeyBase()`.

To set the global key, invoke the `chain.SetSecretKeyBase(key string)` method. Key should be either 16, 24, or 32
bytes to select AES-128, AES-192, or AES-256.

In order to allow key rotation, whenever `SetSecretKeyBase` is called the previous key is stored and available for
decryption of values and the new key will be set as the primary key.

To get the current primary key, invoke the `chain.SecretKeyBase()` method.

To get all the global keys that have already been set (rotation), invoke the `chain.SecretKeys()` method.

If you are interested in being informed every time the `SecretKeyBase` changes, use
the `chain.SecretKeySync(sync SecretKeySyncFunc) (cancel func())` method.

## Keyring

Keyring provides a mechanism that facilitates the management of keys by facilitating rotation.

To create a Keyring that uses `SecretKeyBase`, use the `chain.NewKeyring("MY_SALT", int, int, string)` method. Keyring
implements shortcut for all other features available in the Crypto package, already solving key rotation.

In the example below, at some point we defined the global `SecretKeyBase` and at another moment we initialized our
Keyring.

From this instance, it is possible to encrypt and decrypt content and messages using the key derived from the global
key.

```go
package main

func moment1() {
	// moment 1, set global key
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}

	aad := []byte("purpose: database key")
	var myKeyring = chain.NewKeyring("SALT", 1000, 32, "sha256")

	encryptedA, _ := myKeyring.Encrypt([]byte("Jack"), aad)
	println(base64.StdEncoding.EncodeToString(encryptedA))
	// output: vQHx4VMqWQwweGl5c5iSrJNNYBVdsGKBQt3JwnFZk5c=
}
```

In a second moment, the global key was changed (some automation of application, for example). Note that this is key
rotation.

With the use of Keyring, from now, all encryptions will be performed using the new derived key, however, we are still
able to decrypt the old content, as Keyring uses the previous keys for decryption in case the new key is not compatible.

```go
package main

func moment2() {
	// moment 2, update global key
	if err := chain.SetSecretKeyBase("fe6d1fed11fa60277fb6a2f73efb8be2"); err != nil {
		panic(err)
	}

	// encrypt using new key
	encryptedB, _ := myKeyring.Encrypt([]byte("Jack"), aad)
	println(base64.StdEncoding.EncodeToString(encryptedB))
	// output: yWBkLMkAVDdIcsThnpYerzy62jU6rRnNHMn+VdLIbBg=

	// decrypt value encrypted by old key
	decryptedA, _ := myKeyring.Decrypt(encryptedA, aad)
	println(string(decryptedA)) // -> Jack

	// decrypt value encrypted by new key
	decryptedB, _ := myKeyring.Decrypt(encryptedB, aad)
	println(string(decryptedB)) // -> Jack
}
```

## KeyGenerator

KeyGenerator uses PBKDF2 (Password-Based Key Derivation Function 2), part of PKCS #5 v2.0 (Password-Based
Cryptography Specification).

It can be used to derive a number of keys for various purposes from a given secret. This lets applications have a
single secure secret, but avoid reusing that key in multiple incompatible contexts.

The returned key is a binary. You may invoke functions in the `base64` module, such as
`base64.StdEncoding.EncodeToString()`, to convert this binary into a textual representation.

See http://tools.ietf.org/html/rfc2898#section-5.2

The `KeyGenerator.Generate` method returns a derived key suitable for use.

```go
secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

cookieSalt := []byte("encrypted cookie")
signedCookieSalt := []byte("signed encrypted cookie")

secret := chain.Crypto().KeyGenerate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
signSecret := chain.Crypto().KeyGenerate(secretKeyBase, signedCookieSalt, 1000, 32, "sha256")

println(base64.StdEncoding.EncodeToString(secret))
// output: hpMv01EYLPyGVlV5cBOJR0eK6HNSHO+zHKMmZp2Ezqw=

println(base64.StdEncoding.EncodeToString(signSecret))
// output: y3/r20tnfIWkRZr4HlaC3GAM4LsvS8KnF0JuIi/G/RQ=
```

## MessageVerifier

`MessageVerifier` makes it easy to generate and verify messages which are signed to prevent tampering.

For example, the [cookie store](https://github.com/syntax-framework/chain/blob/main/middlewares/session/store_cookie.go)
uses this verifier to send data to the client. The data can be read by the client, but cannot be tampered with.

The message and its verification are base64url encoded and returned to you.

This is useful for cases like remember-me tokens and auto-unsubscribe links.

```go
message := []byte("This is content")
secret := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

signed := chain.Crypto().MessageSign(secret, message, "sha256")
println(signed)
// output: SFMyNTY.VGhpcyBpcyBjb250ZW50.m-DwbnWabePV8K7-lUNhS8c6gWnwpQcAAhaQ6V2fwA8

verified, _ := chain.Crypto().MessageVerify(secret, []byte(signed))
println(string(verified))
// output: This is content
```

#### Decoding using javascript

MessageVerifier does not encrypt your data, it only signs it. In this way, the signed message can be easily read via
javascript, making it an excellent mechanism to share data with the frontend with the guarantee that the value cannot be
modified.

```javascript
let signed = 'SFMyNTY.VGhpcyBpcyBjb250ZW50.m-DwbnWabePV8K7-lUNhS8c6gWnwpQcAAhaQ6V2fwA8';
let content = atob(signed.split('.')[1])
console.log(content); // This is content
```

## MessageEncryptor

`MessageEncryptor` is a simple way to encrypt values which get stored somewhere you don't trust.

The encrypted key is base64url encoded and returned to you.

This can be used in situations similar to the `MessageVerifier`, but where you don't want users to be able to determine
the value of the payload.

```go
data := []byte("This is content")

secretKeyBase := []byte("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy")

cookieSalt := []byte("encrypted cookie")

encryptionKey := chain.Crypto().KeyGenerate(secretKeyBase, cookieSalt, 1000, 32, "sha256")
aad := []byte("purpose: database key")

encrypted, _ := chain.Crypto().MessageEncrypt(encryptionKey, data, aad)
println(encrypted)
// output: QTEyOEdDTQ.lf2BBZ_rkL6-hBJvW2-qXOgDWtHsDK5UXbhjpHJsoK_BfDjVTocmOsspapQ.r5D76APXc8U7ZBTCv-Cci8rCPFwZCW_hXY2D19rjMkmyc-1kBeYZtXaTgQ

decrypted, _ := chain.Crypto().MessageDecrypt(encryptionKey, []byte(encrypted), aad)
println(string(decrypted))
// output: This is content
```

> Note that, unlike `MessageVerifier`, the result (`encrypted`) cannot be read. Only in possession of `secret`
> and `signSecred` is it possible to access the original content.

## More about Crypto

- [`/examples/crypto`](../examples/crypto)
- [`/crypto`](../crypto)
