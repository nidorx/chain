package session

import (
	"github.com/nidorx/chain"
	"net/http"
	"time"
)

// Config cookie store expects conn.secret_key_base to be set
type Config struct {
	Key        string        // session cookie key (required)
	Path       string        // see http.Cookie
	Domain     string        // see http.Cookie
	Expires    time.Time     // see http.Cookie
	RawExpires string        // see http.Cookie
	MaxAge     int           // see http.Cookie
	Secure     bool          // see http.Cookie
	HttpOnly   bool          // see http.Cookie
	SameSite   http.SameSite // see http.Cookie
	Raw        string        // see http.Cookie
	Unparsed   []string      // see http.Cookie
}

// Store Specification for session stores.
type Store interface {
	Name() string

	// Init Initializes the store.
	Init(config Config, router *chain.Router) error

	// Get Parses the given cookie.
	//
	// Returns a session id and the session contents. The session id is any value that can be used to identify the
	// session by the store.
	//
	// The session id may be nil in case the cookie does not identify any value in the store. The session contents must
	// be a map.
	Get(ctx *chain.Context, rawCookie string) (sid string, data map[string]any)

	// Put  Stores the session associated with given session id.
	//
	// If an empty string is given as sid, a new session id should be generated and returned.
	Put(ctx *chain.Context, sid string, data map[string]any) (rawCookie string, err error)

	// Delete Removes the session associated with given session id from the store.
	Delete(ctx *chain.Context, sid string)
}
