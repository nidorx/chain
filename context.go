package chain

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type chainContextKey struct{}

// ContextKey is the request context key under which URL params are stored.
var ContextKey = chainContextKey{}

// GetContext pulls the URL parameters from a request context, or returns nil if none are present.
func GetContext(ctx context.Context) *Context {
	p, _ := ctx.Value(ContextKey).(*Context)
	return p
}

// Context represents a request & response Context.
type Context struct {
	paramCount        int
	pathSegmentsCount int
	pathSegments      [32]int
	path              string
	paramNames        [32]string
	paramValues       [32]string
	data              map[interface{}]interface{}
	handler           Handle
	router            *Router
	MatchedRoutePath  string
	Writer            http.ResponseWriter
	Request           *http.Request
	Crypto            *chainCrypto

	// A secret key used to verify and encrypt cookies.
	//
	// The field must be set manually whenever one of those features are used.
	//
	// This data must be kept in the connection and never used directly, always use router.Crypto.keyGenerator.Generate()
	// to derive keys from it
	SecretKeyBase string
}

// Set define um valor compartilhado no contexto de execução da requisição
func (ctx *Context) Set(key interface{}, value interface{}) {
	if ctx.data == nil {
		ctx.data = make(map[interface{}]interface{})
	}
	ctx.data[key] = value
}

// Get obtém um valor compartilhado no contexto de execução da requisição
func (ctx *Context) Get(key interface{}) interface{} {
	if ctx.data == nil {
		return nil
	}
	if value, exists := ctx.data[key]; exists {
		return value
	}
	return nil
}

// GetParam returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ctx *Context) GetParam(name string) string {
	for i := 0; i < ctx.paramCount; i++ {
		if ctx.paramNames[i] == name {
			return ctx.paramValues[i]
		}
	}
	return ""
}

// GetParamByIndex get one parameter per index
func (ctx *Context) GetParamByIndex(index int) string {
	return ctx.paramValues[index]
}

// addParameter adds a new parameter to the Context.
func (ctx *Context) addParameter(name string, value string) {
	ctx.paramNames[ctx.paramCount] = name
	ctx.paramValues[ctx.paramCount] = value
	ctx.paramCount++
}

func (ctx *Context) WithParams(names []string, values []string) *Context {
	var context *Context
	if ctx.router != nil {
		context = ctx.router.getContext(ctx.Request, ctx.Writer, "")
	} else {
		context = &Context{
			Writer:      ctx.Writer,
			Request:     ctx.Request,
			handler:     ctx.handler,
			paramCount:  len(names),
			paramNames:  ctx.paramNames,
			paramValues: ctx.paramValues,
		}
	}
	for i := 0; i < len(names); i++ {
		context.paramNames[i] = names[i]
		context.paramValues[i] = values[i]
	}
	return context
}

func (ctx *Context) Dispose() {
	if ctx.router != nil {
		ctx.router.putContext(ctx)
	}
}

// Header returns the header map that will be sent by
// WriteHeader. The Header map also is the mechanism with which
// Handlers can set HTTP trailers.
//
// Changing the header map after a call to WriteHeader (or
// Write) has no effect unless the HTTP status code was of the
// 1xx class or the modified headers are trailers.
//
// There are two ways to set Trailers. The preferred way is to
// predeclare in the headers which trailers you will later
// send by setting the "Trailer" header to the names of the
// trailer keys which will come later. In this case, those
// keys of the Header map are treated as if they were
// trailers. See the example. The second way, for trailer
// keys not known to the Handle until after the first Write,
// is to prefix the Header map keys with the TrailerPrefix
// constant value. See TrailerPrefix.
//
// To suppress automatic response headers (such as "Date"), set
// their value to nil.
func (ctx *Context) Header() http.Header {
	return ctx.Writer.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType. Additionally, if the total size of all written
// data is under a few KB and there are no Flush calls, the
// Content-Length header is added automatically.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (ctx *Context) Write(data []byte) (int, error) {
	return ctx.Writer.Write(data)
}

// WriteHeader sends an HTTP response header with the provided
// status code.
//
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes or 1xx informational responses.
//
// The provided code must be a valid HTTP 1xx-5xx status code.
// Any number of 1xx headers may be written, followed by at most
// one 2xx-5xx header. 1xx headers are sent immediately, but 2xx-5xx
// headers may be buffered. Use the Flusher interface to send
// buffered data. The header map is cleared when 2xx-5xx headers are
// sent, but not with 1xx headers.
//
// The server will automatically send a 100 (Continue) header
// on the first read from the request body if the request has
// an "Expect: 100-continue" header.
func (ctx *Context) WriteHeader(statusCode int) {
	ctx.Writer.WriteHeader(statusCode)
}

// SetCookie adds a Set-Cookie header to the provided ResponseWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be silently dropped.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.Writer, cookie)
}

// DeleteCookie delete a cookie by name
func (ctx *Context) DeleteCookie(name string) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now(),
		MaxAge:   -1,
	})
}

// GetCookie returns the named cookie provided in the request or nil if not found.
// If multiple cookies match the given name, only one cookie will be returned.
func (ctx *Context) GetCookie(name string) *http.Cookie {
	if cookie, err := ctx.Request.Cookie(name); err != nil {
		return cookie
	}
	return nil
}

// RegisterBeforeSend Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (ctx *Context) RegisterBeforeSend(callback func()) error {
	if spy, is := ctx.Writer.(*ResponseWriterSpy); is {
		return spy.registerBeforeSend(callback)
	}
	return nil
}

func (ctx *Context) parsePathSegments() {
	var (
		segmentStart = 0
		segmentSize  int
		path         = ctx.path
	)
	if len(path) > 0 {
		path = path[1:]
	}

	ctx.pathSegments[0] = 0
	ctx.pathSegmentsCount = 1

	for {
		segmentSize = strings.IndexByte(path, separator)
		if segmentSize == -1 {
			segmentSize = len(path)
		}
		ctx.pathSegments[ctx.pathSegmentsCount] = segmentStart + 1 + segmentSize

		if segmentSize == len(path) {
			break
		}
		ctx.pathSegmentsCount++
		path = path[segmentSize+1:]
		segmentStart = segmentStart + 1 + segmentSize
	}
}
