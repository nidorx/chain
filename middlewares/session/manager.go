package session

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nidorx/chain"
)

var globalManagers = map[*chain.Router]*Manager{}

var (
	sessionKey     = "syntax.chain.session."         // Session on chain.Context
	managerKey     = "syntax.chain.session-manager." // Manager on chain.Context
	ErrCannotFetch = errors.New("cannot fetch session, check if there is a session.Manager configured")
)

// Manager cookie store expects conn.secret_key_base to be set
type Manager struct {
	Config
	Store Store // session store module (required)
}

func (m *Manager) Init(method string, path string, router *chain.Router) {

	if m.Store == nil {
		m.Store = &Cookie{}
		panic(fmt.Sprintf("[chain.middlewares.session] store is required. Method: %s, Path: %s", method, path))
	}
	if strings.TrimSpace(m.Key) == "" {
		panic(fmt.Sprintf("[chain.middlewares.session] key is required. Method: %s, Path: %s", method, path))
	}

	if (method == "" || method == "*") && (path == "" || path == "*" || path == "/*") {
		if _, exist := globalManagers[router]; exist {
			panic(fmt.Sprintf("[chain.middlewares.session] there is already a global session.Manager registered for this chain.Router. Method: %s, Path: %s", method, path))
		}
		globalManagers[router] = m
	}

	if err := m.Store.Init(m.Config, router); err != nil {
		panic(fmt.Sprintf("[chain.middlewares.session] error initializing store. store: %s", m.Store.Name()))
	}
}

func (m *Manager) Handle(ctx *chain.Context, next func() error) error {
	ctx.Set(managerKey+m.Key, m)
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
	ctx.Set(sessionKey+m.Key, session)
	if err := ctx.BeforeSend(func() { m.beforeSend(ctx, sid, session) }); err != nil {
		return nil, err
	}
	return session, nil
}

func (m *Manager) beforeSend(ctx *chain.Context, sid string, session *Session) {
	switch session.state {
	case write:
		rawCookie, err := m.Store.Put(ctx, sid, session.data)
		if err != nil {
			slog.Error(
				"[chain.middlewares.session] error saving session in store",
				slog.Any("Error", err),
				slog.String("Store", m.Store.Name()),
			)
		} else {
			m.setCookie(ctx, rawCookie)
		}
	case drop:
		if sid != "" {
			m.Store.Delete(ctx, sid)
			ctx.RemoveCookie(m.Key)
		}
	case renew:
		if sid != "" {
			m.Store.Delete(ctx, sid)
		}
		rawCookie, err := m.Store.Put(ctx, "", session.data)
		if err != nil {
			slog.Error(
				"[chain.middlewares.session] error saving session in store",
				slog.Any("Error", err),
				slog.String("Store", m.Store.Name()),
			)
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

// FetchByKey LazyLoad session from context using a session.Manager Key
func FetchByKey(ctx *chain.Context, key string) (*Session, error) {
	if value, exist := ctx.Get(sessionKey + key); exist && value != nil {
		if session, valid := value.(*Session); valid {
			return session, nil
		}
	}

	if value, exist := ctx.Get(managerKey + key); exist && value != nil {
		if manager, valid := value.(*Manager); valid {
			return manager.fetch(ctx)
		}
	}

	return nil, ErrCannotFetch
}

// Fetch LazyLoad session from context. It only returns result if there is a global session.Manager configured
func Fetch(ctx *chain.Context) (*Session, error) {
	router := ctx.Router()
	if manager, exist := globalManagers[router]; exist {
		return manager.fetch(ctx)
	}

	return nil, ErrCannotFetch
}
