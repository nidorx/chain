package session

import (
	"errors"
	"github.com/syntax-framework/chain"
	"net/http"
	"strings"
	"time"
)

type privateManagerKey struct{}

var (
	sessionKey     = "syntax.chain.session" // Session on chain.Context
	managerKey     = privateManagerKey{}    // Manager on chain.Context
	ErrCannotFetch = errors.New("cannot fetch session, check if there is a session.Manager configured")
)

// Manager cookie store expects conn.secret_key_base to be set
type Manager struct {
	Config
	Store Store // session store module (required)
}

// @TODO: func FetchByKey(ctx *chain.Context, key string) (*Session, error)

// Fetch LazyLoad session from context
func Fetch(ctx *chain.Context) (*Session, error) {

	if value := ctx.Get(sessionKey); value != nil {
		if session, valid := value.(*Session); valid {
			return session, nil
		}
	}

	if value := ctx.Get(managerKey); value != nil {
		if manager, valid := value.(*Manager); valid {
			return manager.fetch(ctx)
		}
	}

	return nil, ErrCannotFetch
}

func (m *Manager) Init(router *chain.Router) {
	if m.Store == nil {
		panic(any("session.Manager: Store is required"))
	}
	if strings.TrimSpace(m.Key) == "" {
		panic(any("session.Manager: Key is required"))
	}

	if err := m.Store.Init(m.Config, router); err != nil {
		panic(any(err))
	}
}

func (m *Manager) Handle(ctx *chain.Context, next func() error) error {
	ctx.Set(managerKey, m)
	return next()
}

// fetch load the session
func (m *Manager) fetch(ctx *chain.Context) (*Session, error) {
	var sid string
	var session *Session

	if cookie := ctx.GetCookie(m.Key); cookie != nil {
		var data map[string]any
		if sid, data = m.Store.Get(ctx, cookie.Value); data == nil {
			data = map[string]any{}
		}
		session = &Session{data: data, state: none}
	} else {
		// new session
		session = &Session{data: map[string]any{}, state: write}
	}
	ctx.Set(sessionKey, session)
	if err := ctx.RegisterBeforeSend(func() { m.beforeSend(ctx, sid, session) }); err != nil {
		return nil, err
	}
	return session, nil
}

func (m *Manager) beforeSend(ctx *chain.Context, sid string, session *Session) {
	switch session.state {
	case write:
		rawCookie, err := m.Store.Put(ctx, sid, session.data)
		if err != nil {
			// @todo: log
			println(err)
		} else {
			m.setCookie(ctx, rawCookie)
		}
	case drop:
		if sid != "" {
			m.Store.Delete(ctx, sid)
			ctx.DeleteCookie(m.Key)
		}
	case renew:
		if sid != "" {
			m.Store.Delete(ctx, sid)
		}
		rawCookie, err := m.Store.Put(ctx, "", session.data)
		if err != nil {
			// @todo: log
			println(err)
		} else {
			m.setCookie(ctx, rawCookie)
		}
	}
}

func (m *Manager) setCookie(ctx *chain.Context, rawCookie string) {
	ctx.SetCookie(&http.Cookie{
		Name:       m.Key,
		Value:      rawCookie,
		Path:       m.Path,
		Domain:     m.Domain,
		Expires:    time.Time{},
		RawExpires: m.RawExpires,
		MaxAge:     m.MaxAge,
		Secure:     m.Secure,
		HttpOnly:   m.HttpOnly,
		SameSite:   m.SameSite,
		Raw:        m.Raw,
		Unparsed:   m.Unparsed,
	})
}
