package socket

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/middlewares/session"
)

const sseSessionId = "_sse_"

type CorsConfig struct {
	MaxAge              time.Duration
	AllowAllOrigins     bool
	AllowCredentials    bool
	AllowPrivateNetwork bool
	AllowOrigins        []string
	AllowOriginFunc     func(string) bool
	AllowMethods        []string
	AllowHeaders        []string
	ExposeHeaders       []string
}

type TransportSSE struct {
	sessionKey string
	Cors       *CorsConfig
	Cookie     *session.Config
}

func (t *TransportSSE) Configure(handler *Handler, router *chain.Router, endpoint string) {
	endpoint = endpoint + "/sse"

	salt := chain.HashMD5(endpoint)
	t.sessionKey = sseSessionId + salt[:8]

	sm := &session.Manager{
		Config: session.Config{
			Key:  t.sessionKey,
			Path: endpoint,
		},
		Store: &session.Cookie{},
	}

	if t.Cookie != nil {
		sm.Config.Domain = t.Cookie.Domain
		sm.Config.MaxAge = t.Cookie.MaxAge
		sm.Config.Secure = t.Cookie.Secure
		sm.Config.HttpOnly = t.Cookie.HttpOnly
		sm.Config.SameSite = t.Cookie.SameSite
	}
	router.Use(endpoint, sm)

	if t.Cors != nil {
		// see: https://github.com/gin-contrib/cors

		maxAge := t.Cors.MaxAge
		allowAllOrigins := t.Cors.AllowAllOrigins
		allowCredentials := t.Cors.AllowCredentials
		allowPrivateNetwork := t.Cors.AllowPrivateNetwork
		allowMethods := strings.Join(t.Cors.AllowMethods, ",")
		allowHeaders := strings.Join(t.Cors.AllowHeaders, ",")
		exposeHeaders := strings.Join(t.Cors.ExposeHeaders, ",")

		router.OPTIONS(endpoint, func(ctx *chain.Context) {
			if len(allowMethods) > 0 {
				ctx.SetHeader("Access-Control-Allow-Methods", allowMethods)
			}
			if len(allowHeaders) > 0 {
				ctx.SetHeader("Access-Control-Allow-Headers", allowHeaders)
			}
			if maxAge > time.Duration(0) {
				value := strconv.FormatInt(int64(maxAge/time.Second), 10)
				ctx.SetHeader("Access-Control-Max-Age", value)
			}

			if allowPrivateNetwork {
				ctx.SetHeader("Access-Control-Allow-Private-Network", "true")
			}

			if allowAllOrigins {
				ctx.SetHeader("Access-Control-Allow-Origin", "*")
			} else {
				// Always set Vary headers
				// see https://github.com/rs/cors/issues/10,
				// https://github.com/rs/cors/commit/dbdca4d95feaa7511a46e6f1efb3b3aa505bc43f#commitcomment-12352001

				ctx.AddHeader("Vary", "Origin")
				ctx.AddHeader("Vary", "Access-Control-Request-Method")
				ctx.AddHeader("Vary", "Access-Control-Request-Headers")
			}
			ctx.WriteHeader(http.StatusNoContent)
		})

		router.Use(endpoint, func(ctx *chain.Context, next func() error) error {
			origin := ctx.Request.Header.Get("Origin")
			if len(origin) == 0 {
				// request is not a CORS request
				return next()
			}
			host := ctx.Request.Host

			if origin == "http://"+host || origin == "https://"+host {
				// request is not a CORS request but have origin header.
				// for example, use fetch api
				return next()
			}

			if !allowAllOrigins {
				isValidOrigin := false
				for _, value := range t.Cors.AllowOrigins {
					if value == origin || value == "*" {
						isValidOrigin = true
						break
					}
				}
				if !isValidOrigin && t.Cors.AllowOriginFunc != nil {
					isValidOrigin = t.Cors.AllowOriginFunc(origin)
				}

				if !isValidOrigin {
					ctx.Forbidden()
					return nil
				}
			}

			if allowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}

			if ctx.Request.Method != "OPTIONS" {
				if len(exposeHeaders) > 0 {
					ctx.SetHeader("Access-Control-Expose-Headers", exposeHeaders)
				}

				if allowAllOrigins {
					ctx.SetHeader("Access-Control-Allow-Origin", "*")
				} else {
					ctx.SetHeader("Vary", "Origin")
				}
			}

			if !allowAllOrigins {
				ctx.SetHeader("Access-Control-Allow-Origin", origin)
			}

			return next()
		})
	}

	// Publish the message.
	router.POST(endpoint, func(ctx *chain.Context) {
		var socketSession *Session
		if socketSession = t.resumeSession(ctx, handler); socketSession == nil {
			ctx.WriteHeader(http.StatusGone)
			return
		}

		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.WriteHeader(http.StatusBadRequest)
			return
		}

		socketSession.Dispatch(body)
		ctx.WriteHeader(http.StatusOK)
	})

	// Starts a new session or listen to a ServiceMsg if one already exists.
	router.GET(endpoint, func(ctx *chain.Context) {
		var ok bool
		var flusher http.Flusher
		if flusher, ok = ctx.Writer.(*chain.ResponseWriterSpy).ResponseWriter.(http.Flusher); !ok {
			ctx.Error("Connection does not support streaming", http.StatusBadRequest)
			return
		}

		var socketSession *Session
		if socketSession = t.resumeSession(ctx, handler); socketSession == nil {
			var err error
			if socketSession, err = t.newSession(handler, ctx, endpoint); err != nil {
				ctx.Error("Could not initialize connection: "+err.Error(), http.StatusForbidden)
				return
			}
		}

		if ctx.Request.ProtoMajor == 1 {
			// An endpoint MUST NOT generate an HTTP/2 message containing connection-specific header fields.
			// Source: RFC7540.
			ctx.SetHeader("Connection", "keep-alive")
		}
		ctx.SetHeader("X-Accel-Buffering", "no")
		ctx.SetHeader("Content-Type", "text/event-stream; charset=utf-8")
		ctx.SetHeader("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0")
		ctx.SetHeader("Pragma", "no-cache")
		ctx.SetHeader("Expire", "0")
		ctx.WriteHeader(http.StatusOK)
		flusher.Flush()
		if err := t.listen(socketSession, ctx, flusher); err != nil {
			ctx.Error(err.Error(), http.StatusInternalServerError)
		}
	})
}

func (t *TransportSSE) resumeSession(ctx *chain.Context, handler *Handler) *Session {
	var sess *session.Session
	var err error
	if sess, err = session.FetchByKey(ctx, t.sessionKey); err != nil {
		return nil
	}

	// browser tab identifier
	sidKey := strings.TrimSpace(ctx.Request.URL.Query().Get("sid"))
	if sidKey == "" {
		sidKey = "sid"
	}

	sid := sess.Get(sidKey)
	if sid == nil {
		return nil
	}

	return handler.Resume(sid.(string))
}

func (t *TransportSSE) newSession(handler *Handler, ctx *chain.Context, endpoint string) (skt *Session, err error) {

	var sess *session.Session
	if sess, err = session.FetchByKey(ctx, t.sessionKey); err != nil {
		return
	}

	params := map[string]string{}
	query := ctx.Request.URL.Query()
	for k := range query {
		params[k] = query.Get(k)
	}

	if skt, err = handler.Connect(endpoint, params); err != nil {
		return
	}

	// browser tab identifier
	sidKey := strings.TrimSpace(ctx.Request.URL.Query().Get("sid"))
	if sidKey == "" {
		sidKey = "sid"
	}
	sess.Put(sidKey, skt.Id())

	return
}

func (t *TransportSSE) listen(socketSession *Session, ctx *chain.Context, flusher http.Flusher) (err error) {

	// after disconnection, schedule session shutdown
	defer socketSession.ScheduleShutdown(time.Second * 15)

	w := ctx.Writer.(*chain.ResponseWriterSpy)

	// trap the request under loop forever
	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case msg := <-socketSession.messages:
			if msg != nil {
				if _, err = fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
