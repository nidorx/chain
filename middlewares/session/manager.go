package session

import (
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/syntax-framework/chain"
	"net/http"
	"strings"
	"time"
)

var globalManagers = map[*chain.Router]*Manager{}

func _l(msg string) string {
	return "[chain.middlewares.session] " + msg
}

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
		log.Panic().Msg(_l("store is required"))
	}
	if strings.TrimSpace(m.Key) == "" {
		log.Panic().Msg(_l("Key is required"))
	}

	if (method == "" || method == "*") && (path == "" || path == "*" || path == "/*") {
		if _, exist := globalManagers[router]; exist {
			log.Panic().Msg(_l("there is already a global session.Manager registered for this chain.Router"))
		}
		globalManagers[router] = m
	}

	if err := m.Store.Init(m.Config, router); err != nil {
		log.Panic().
			Str("store", m.Store.Name()).
			Msg(_l("error initializing store"))
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
			log.Error().Err(err).
				Caller(0).
				Str("store", m.Store.Name()).
				Msg(_l("error saving session in store"))
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
			log.Error().Err(err).
				Caller(0).
				Str("store", m.Store.Name()).
				Msg(_l("error saving session in store"))
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
	if value := ctx.Get(sessionKey + key); value != nil {
		if session, valid := value.(*Session); valid {
			return session, nil
		}
	}

	if value := ctx.Get(managerKey + key); value != nil {
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
