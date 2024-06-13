package chain

import (
	"bytes"
	"net/http"
	"strconv"
	"time"
)

var UnixEpoch = time.Unix(0, 0)
var jsonSerializer = &JsonSerializer{}

// Json encode and writes the data to the connection as part of an HTTP reply.
//
// The Content-Length and Content-Type headers are added automatically.
func (ctx *Context) Json(v any) {
	if encoded, err := jsonSerializer.Encode(v); err != nil {
		ctx.Error(err.Error(), http.StatusInternalServerError)
	} else {
		ctx.SetHeader("Content-Type", "application/json")
		ctx.ServeContent(encoded, "", UnixEpoch)
	}
}

// WriteStarted returns true if the ctx.Writer.Write or ctx.Writer.WriteHeader method was called
func (ctx *Context) WriteStarted() bool {
	if w, ok := ctx.Writer.(*ResponseWriterSpy); ok {
		return w.writeStarted
	}
	return true
}

// WriteCalled returns true if the ctx.Writer.Write method was called
func (ctx *Context) WriteCalled() bool {
	if w, ok := ctx.Writer.(*ResponseWriterSpy); ok {
		return w.writeCalled
	}
	return true
}

// WriteCalled returns true if the ctx.Writer.WriteHeader method was called
func (ctx *Context) WriteHeaderCalled() bool {
	if w, ok := ctx.Writer.(*ResponseWriterSpy); ok {
		return w.writeHeaderCalled
	}
	return true
}

// ServeContent replies to the request using the content in the
// provided. The main benefit of ServeContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
//
// If the response's Content-Type header is not set, ServeContent
// first tries to deduce the type from name's file extension and,
// if that fails, falls back to reading the first block of the content
// and passing it to DetectContentType.
// The name is otherwise unused; in particular it can be empty and is
// never sent in the response.
//
// If modtime is not the zero time or Unix epoch, ServeContent
// includes it in a Last-Modified header in the response. If the
// request includes an If-Modified-Since header, ServeContent uses
// modtime to decide whether the content needs to be sent at all.
//
// The content's Seek method must work: ServeContent uses
// a seek to the end of the content to determine its size.
//
// If the caller has set w's ETag header formatted per RFC 7232, section 2.3,
// ServeContent uses it to handle requests using If-Match, If-None-Match, or If-Range.
func (ctx *Context) ServeContent(content []byte, name string, modtime time.Time) {
	ctx.SetHeader("ETag", HashXxh64(content))
	ctx.SetHeader("Content-Length", strconv.Itoa(len(content)))
	http.ServeContent(ctx.Writer, ctx.Request, name, modtime, bytes.NewReader(content))
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

// SetHeader sets the header entries associated with key to the single element value. It replaces any existing values
// associated with key. The key is case insensitive; it is canonicalized by textproto.CanonicalMIMEHeaderKey.
// To use non-canonical keys, assign to the map directly.
func (ctx *Context) SetHeader(key, value string) {
	ctx.Writer.Header().Set(key, value)
}

// AddHeader adds the key, value pair to the header.
// It appends to any existing values associated with key.
// The key is case insensitive; it is canonicalized by CanonicalHeaderKey.
func (ctx *Context) AddHeader(key, value string) {
	ctx.Writer.Header().Add(key, value)
}

// ContentType set the response content type
func (ctx *Context) ContentType(ctype string) {
	ctx.SetHeader("Content-Type", ctype)
}

// Redirect replies to the request with a redirect to url, which may be a path relative to the request path.
//
// The provided code should be in the 3xx range and is usually StatusMovedPermanently, StatusFound or StatusSeeOther.
//
// If the Content-Type header has not been set, Redirect sets it to "text/html; charset=utf-8" and writes a small HTML
// body.
//
// Setting the Content-Type header to any value, including nil, disables that behavior.
func (ctx *Context) Redirect(url string, code int) {
	http.Redirect(ctx.Writer, ctx.Request, url, code)
}

// SetCookie adds a Set-Cookie header to the provided ResponseWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be silently dropped.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.Writer, cookie)
}

// RemoveCookie delete a cookie by name
func (ctx *Context) RemoveCookie(name string) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now(),
		MaxAge:   -1,
	})
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

// WriteHeader sends an HTTP response header with the provided status code.
func (ctx *Context) Status(statusCode int) {
	ctx.WriteHeader(statusCode)
}

// Created sends an HTTP response header with the 200 OK status code.
func (ctx *Context) OK() {
	ctx.Status(http.StatusOK)
}

// Created sends an HTTP response header with the 201 Created status code.
func (ctx *Context) Created() {
	ctx.Status(http.StatusCreated)
}

// Created sends an HTTP response header with the 204 No Content status code.
func (ctx *Context) NoContent() {
	ctx.Status(http.StatusNoContent)
}

// Error replies to the request with the specified error message and HTTP code.
// It does not otherwise end the request; the caller should ensure no further writes are done to w.
// The error message should be plain text.
func (ctx *Context) Error(error string, code int) {
	http.Error(ctx.Writer, error, code)
}

// BadRequest replies to the request with an HTTP 400 bad request error.
func (ctx *Context) BadRequest() {
	ctx.Error("400 Bad Request", http.StatusBadRequest)
}

// Unauthorized replies to the request with an HTTP 401 Unauthorized error.
func (ctx *Context) Unauthorized() {
	ctx.Error("401 Unauthorized", http.StatusUnauthorized)
}

// Unauthorized replies to the request with an HTTP 403 Forbidden error.
func (ctx *Context) Forbidden() {
	ctx.Error("403 Forbidden", http.StatusForbidden)
}

// NotFound replies to the request with an HTTP 404 not found error.
func (ctx *Context) NotFound() {
	http.NotFound(ctx.Writer, ctx.Request)
}

// TooManyRequests replies to the request with an HTTP 429 Too Many Requests error.
func (ctx *Context) TooManyRequests() {
	ctx.Error("429 Too Many Requests", http.StatusTooManyRequests)
}

// InternalServerError replies to the request with an HTTP 500 Internal Server Error error.
func (ctx *Context) InternalServerError() {
	ctx.Error("500 Internal Server Error", http.StatusInternalServerError)
}

// NotImplemented replies to the request with an HTTP 501 Not Implemented error.
func (ctx *Context) NotImplemented() {
	ctx.Error("501 Not Implemented", http.StatusNotImplemented)
}

// ServiceUnavailable replies to the request with an HTTP 503 Service Unavailable error.
func (ctx *Context) ServiceUnavailable() {
	ctx.Error("503 Service Unavailable", http.StatusServiceUnavailable)
}
