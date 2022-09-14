// Stores the session in a cookie.
//
// This cookie store is based on `chain.MessageVerifier` and `chain.MessageEncryptor` which encrypts and signs
// each cookie to ensure they can't be read nor tampered with.
//
// Since this store uses crypto features, it requires you to set the `SecretKeyBase` field in your router. This
// can be easily achieved with:
//
//	router := chain.New()
//	router.SecretKeyBase = "-- LONG STRING WITH AT LEAST 64 BYTES --"
//
// ## Example
//
//	router := chain.New()
//	router.Use(session.Manager{
//		Store: session.Cookie{
//	    	Key: 			"_my_app_session",
//	        EncryptionSalt: "cookie store encryption salt",
//	        SigningSalt: 	"cookie store signing salt",
//	        KeyLength: 		64,
//	        Log: 			"debug"
//		}
//	})

package session

import (
	"encoding/json"
	"errors"
	"github.com/syntax-framework/chain"
	"strings"
)

type Serializer interface {
	Encode(v map[string]any) ([]byte, error)
	Decode(data []byte) (map[string]any, error)
}

type JsonSerializer struct {
}

func (s *JsonSerializer) Encode(v map[string]any) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JsonSerializer) Decode(data []byte) (map[string]any, error) {
	output := map[string]any{}
	if err := json.Unmarshal(data, output); err != nil {
		return nil, err
	}
	return output, nil
}

var ErrSecretKeyBaseEmpty = errors.New("cookie store expects SecretKeyBase or router.SecretKeyBase to be set")
var ErrSecretKeyBaseLen = errors.New("cookie store expects SecretKeyBase to be at least 64 bytes")
var defaultSerializer = &JsonSerializer{}

type CryptoOptions struct {
	SecretKeyBase  string // the secret key base to built the cookie signing/encryption on top of.
	EncryptionSalt string // a salt used with `SecretKeyBase` to generate a key for encrypting/decrypting a cookie.
	SigningSalt    string // a salt used with `SecretKeyBase` to generate a key for signing/verifying a cookie.
	Iterations     int    // option passed to `chain.keyGenerator` when generating the encryption and signing keys. Defaults to 1000;
	Length         int    // option passed to `chain.keyGenerator` when generating the encryption and signing keys. Defaults to 32;
	Digest         string // option passed to `chain.keyGenerator` when generating the encryption and signing keys. Defaults to `sha256`;
}

// Cookie Stores the session in a cookie.
// https://edgeapi.rubyonrails.org/classes/ActionDispatch/Session/CookieStore.html
// https://funcptr.net/2013/08/25/user-sessions,-what-data-should-be-stored-where-/
type Cookie struct {
	CryptoOptions
	Serializer      Serializer      // cookie serializer module that defines `Encode(any)` and `Decode(any)`. Defaults to `json`.
	Log             string          // Log level to use when the cookie cannot be decoded. Defaults to `debug`, can be set to false to disable it.
	RotatingOptions []CryptoOptions // additional list of options to use when decrypting and verifying the cookie. These options
	//  are used only when the cookie could not be decoded using primary options and are fetched on init so they cannot be
	//  changed in runtime. Defaults to `[]`.
}

func (c *Cookie) Name() string { return "Cookie" }

func (c *Cookie) Init(config Config, router *chain.Router) (err error) {

	c.SecretKeyBase = strings.TrimSpace(c.SecretKeyBase)
	if c.SecretKeyBase == "" {
		// get from chain.SecretKeyBase
		c.SecretKeyBase = strings.TrimSpace(router.SecretKeyBase)
	}
	if err = validateSecretKeyBase(c.SecretKeyBase); err != nil {
		return
	}

	if c.Iterations == 0 {
		c.Iterations = 1000
	}

	if c.Length == 0 {
		c.Length = 32
	}

	if strings.TrimSpace(c.Digest) == "" {
		c.Digest = "sha256"
	}

	if strings.TrimSpace(c.Log) == "" {
		c.Log = "debug"
	}

	if c.Serializer == nil {
		c.Serializer = defaultSerializer
	}

	// pre derive
	if strings.TrimSpace(c.SigningSalt) == "" {
		panic(any("cookie store expects SigningSalt"))
	}
	var signingSalt []byte
	if signingSalt, err = c.derive(c.SecretKeyBase, c.SigningSalt, &c.CryptoOptions); err != nil {
		return
	}
	c.SigningSalt = string(signingSalt)

	c.EncryptionSalt = strings.TrimSpace(c.EncryptionSalt)
	if c.EncryptionSalt != "" {
		// pre derive
		var encryptionSalt []byte
		if encryptionSalt, err = c.derive(c.SecretKeyBase, c.EncryptionSalt, &c.CryptoOptions); err != nil {
			return
		}
		c.EncryptionSalt = string(encryptionSalt)
	}

	return
}

func (c *Cookie) Get(ctx *chain.Context, rawCookie string) (sid string, data map[string]any) {

	options := []CryptoOptions{
		{
			SecretKeyBase:  ctx.SecretKeyBase,
			EncryptionSalt: c.EncryptionSalt,
			SigningSalt:    c.SigningSalt,
			Iterations:     c.Iterations,
			Length:         c.Length,
			Digest:         c.Digest,
		},
	}

	if len(c.RotatingOptions) > 0 {
		options = append(options, c.RotatingOptions...)
	}

	var (
		err        error
		serialized []byte
		binary     = []byte(rawCookie)
	)
	for _, option := range options {
		if serialized, err = c.readRawCookie(binary, &option); err != nil {
			continue
		}
		if data, err = c.Serializer.Decode(serialized); err != nil {
			println(err)
		}
		break
	}
	return
}

func (c *Cookie) Put(ctx *chain.Context, sid string, data map[string]any) (rawCookie string, err error) {
	var encoded []byte
	if encoded, err = c.Serializer.Encode(data); err != nil {
		return
	}

	var signingSalt []byte
	if signingSalt, err = c.derive(c.SecretKeyBase, c.SigningSalt, &c.CryptoOptions); err != nil {
		return
	}
	if c.EncryptionSalt == "" {
		rawCookie = chain.MessageVerifier.Sign(encoded, signingSalt, c.Digest)
	} else {
		var encryptionSalt []byte
		if encryptionSalt, err = c.derive(c.SecretKeyBase, c.EncryptionSalt, &c.CryptoOptions); err != nil {
			return
		}
		rawCookie, err = chain.MessageEncryptor.Encrypt(encoded, encryptionSalt, signingSalt)
	}

	return
}

func (c *Cookie) Delete(ctx *chain.Context, sid string) {}

func (c *Cookie) readRawCookie(rawCookie []byte, opts *CryptoOptions) (serialized []byte, err error) {

	var signingSalt []byte
	if signingSalt, err = c.derive(opts.SecretKeyBase, opts.SigningSalt, opts); err != nil {
		return
	}

	if opts.EncryptionSalt == "" {
		return chain.MessageVerifier.Verify(rawCookie, signingSalt)
	} else {
		var encryptionSalt []byte
		if encryptionSalt, err = c.derive(opts.SecretKeyBase, opts.EncryptionSalt, opts); err != nil {
			return nil, err
		}
		return chain.MessageEncryptor.Decrypt(rawCookie, encryptionSalt, signingSalt)
	}
}

func (c *Cookie) derive(secretKeyBase string, salt string, opts *CryptoOptions) (derived []byte, err error) {
	if err = validateSecretKeyBase(secretKeyBase); err != nil {
		return
	}
	derived = chain.KeyGenerator.Generate([]byte(secretKeyBase), []byte(salt), opts.Iterations, opts.Length, opts.Digest)
	return
}

func validateSecretKeyBase(SecretKeyBase string) error {
	if SecretKeyBase == "" {
		return ErrSecretKeyBaseEmpty
	}

	if len(SecretKeyBase) < 8 {
		return ErrSecretKeyBaseLen
	}
	return nil
}
