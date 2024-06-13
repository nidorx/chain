package crypto

type Crypto interface {
	Decrypt(secret []byte, encrypted []byte, aad []byte) (plain []byte, err error)
	Encrypt(secret []byte, data []byte, aad []byte) (encrypted []byte, err error)
	KeyGenerate(secret []byte, salt []byte, iterations int, length int, digest string) []byte
	MessageDecrypt(secret []byte, encoded []byte, aad []byte) (content []byte, err error)
	MessageEncrypt(secret []byte, content []byte, aad []byte) (encoded string, err error)
	MessageSign(secret []byte, message []byte, digest string) string
	MessageVerify(secret []byte, signed []byte) (decoded []byte, err error)
}
