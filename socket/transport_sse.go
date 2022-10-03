package socket

import (
	"fmt"
	"github.com/syntax-framework/chain"
	"github.com/syntax-framework/chain/middlewares/session"
	"io"
	"net/http"
	"time"
)

const sseSessionId = "_sse_"

type TransportSSE struct {
	sessionKey string
	keyBase    string
	keySalt    string
}

func (t *TransportSSE) Configure(handler *Handler, router *chain.Router, endpoint string) {
	endpoint = endpoint + "/sse"

	salt := chain.HashMD5(endpoint)
	t.sessionKey = sseSessionId + salt[:8]
	// does not save sensitive information
	t.keyBase = chain.HashMD5(t.sessionKey)
	t.keySalt = string(chain.Crypto.Generator.Generate([]byte(t.keyBase), []byte(salt), 0, 0, ""))

	// @todo: validate the Origin header

	router.Use(endpoint, &session.Manager{
		Config: session.Config{
			Key:  t.sessionKey,
			Path: endpoint,
		},
		Store: &session.Cookie{
			CryptoOptions: session.CryptoOptions{
				SecretKeyBase: t.keyBase,
				SigningSalt:   t.keySalt,
			},
		},
	})

	// Publish the message.
	router.POST(endpoint, func(ctx *chain.Context) {
		var worker *Session
		if worker = t.resumeSession(ctx, handler); worker == nil {
			ctx.WriteHeader(http.StatusGone)
			return
		}

		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.WriteHeader(http.StatusBadRequest)
			return
		}

		worker.Dispatch(body)
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

		var worker *Session
		if worker = t.resumeSession(ctx, nil); worker == nil {
			var err error
			if worker, err = t.newSession(handler, ctx, endpoint); err != nil {
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
		//ctx.SetHeader("Access-Control-Allow-Origin", "*")
		//ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST")
		//ctx.SetHeader("Access-Control-Allow-Headers", ctx.GetHeader("Access-Control-Request-Headers"))
		//ctx.SetHeader("Access-Control-Max-Age", "3600")
		//ctx.SetHeader("Cache-Control", "no-cache")
		//ctx.SetHeader("Access-Control-Allow-Origin", "*")
		ctx.WriteHeader(http.StatusOK)
		flusher.Flush()
		if err := t.listen(worker, ctx, flusher); err != nil {
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
	sid := sess.Get("sid")
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
	for k, _ := range query {
		params[k] = query.Get(k)
	}

	if skt, err = handler.Connect(endpoint, params); err != nil {
		return
	}
	sess.Put("sid", skt.SocketId())

	return
}

func (t *TransportSSE) listen(session *Session, ctx *chain.Context, flusher http.Flusher) (err error) {

	// after disconnection, schedule session shutdown
	defer session.ScheduleShutdown(time.Second * 15)

	w := ctx.Writer.(*chain.ResponseWriterSpy)

	// trap the request under loop forever
	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case msg := <-session.messages:
			if msg != nil {
				if _, err = fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
