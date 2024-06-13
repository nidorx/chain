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
//	        Log: 			"debug"
//		}
//	})

package session

import (
	"log/slog"
	"strings"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/crypto"
)

var (
	defaultSerializer     = &chain.JsonSerializer{}
	defaultSigningKeyring = chain.NewKeyring("chain.middleware.session.keyring.salt", 1000, 32, "sha256")
	defaultEncryptionAAD  = []byte("chain.middleware.session.cookie.aad")
)

// Cookie Stores the session in a cookie.
// https://edgeapi.rubyonrails.org/classes/ActionDispatch/Session/CookieStore.html
// https://funcptr.net/2013/08/25/user-sessions,-what-data-should-be-stored-where-/
type Cookie struct {
	Log               string           // Log level to use when the cookie cannot be decoded. Defaults to `debug`, can be set to false to disable it.
	Serializer        chain.Serializer // cookie serializer module that defines `Encode(any)` and `Decode(any)`. Defaults to `json`.
	SigningKeyring    *crypto.Keyring  // a crypto.Keyring used with for signing/verifying a cookie.
	EncryptionKeyring *crypto.Keyring  // a crypto.Keyring used for encrypting/decrypting a cookie.
	EncryptionAAD     []byte           // Additional authenticated data (AAD)
}

func (c *Cookie) Name() string { return "Cookie" }

func (c *Cookie) Init(config Config, router *chain.Router) (err error) {

	if c.SigningKeyring == nil {
		c.SigningKeyring = defaultSigningKeyring
	}

	if strings.TrimSpace(c.Log) == "" {
		c.Log = "debug"
	}

	if c.Serializer == nil {
		c.Serializer = defaultSerializer
	}

	return
}

func (c *Cookie) Get(ctx *chain.Context, rawCookie string) (sid string, data map[string]any) {
	var (
		err        error
		serialized []byte
		binary     = []byte(rawCookie)
	)

	if c.EncryptionKeyring == nil {
		serialized, err = c.SigningKeyring.MessageVerify(binary)
	} else {
		aad := defaultEncryptionAAD
		if c.EncryptionAAD != nil {
			aad = c.EncryptionAAD
		}
		serialized, err = c.EncryptionKeyring.MessageDecrypt(binary, aad)
	}

	if err == nil {
		var decoded any
		if decoded, err = c.Serializer.Decode(serialized, &map[string]any{}); err == nil {
			data = *decoded.(*map[string]any)
			return
		}
	}

	slog.Debug(
		"[chain.middlewares.session] could not decode serialized data",
		slog.Any("Error", err),
		slog.Any("Store", c.Name()),
	)
	return
}

func (c *Cookie) Put(ctx *chain.Context, sid string, data map[string]any) (rawCookie string, err error) {
	var encoded []byte
	if encoded, err = c.Serializer.Encode(data); err != nil {
		return
	}

	if c.EncryptionKeyring == nil {
		rawCookie, err = c.SigningKeyring.MessageSign(encoded, "sha256")
	} else {
		aad := defaultEncryptionAAD
		if c.EncryptionAAD != nil {
			aad = c.EncryptionAAD
		}
		rawCookie, err = c.EncryptionKeyring.MessageEncrypt(encoded, aad)
	}

	return
}

func (c *Cookie) Delete(ctx *chain.Context, sid string) {}
