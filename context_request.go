package chain

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// BodyBytes get body as array of bytes.
// It respects the MaxBodySize limit if set on the context.
func (ctx *Context) BodyBytes() (body []byte, err error) {
	if cb, exist := ctx.Get(BodyBytesKey); exist && cb != nil {
		if cbb, ok := cb.([]byte); ok {
			body = cbb
		}
	}

	if body == nil {
		// Check content length against max body size
		if ctx.Request.ContentLength > 0 {
			maxSize := ctx.getMaxBodySize()
			if ctx.Request.ContentLength > maxSize {
				return nil, fmt.Errorf("%w: content-length %d exceeds maximum %d bytes", ErrRequestBodyTooLarge, ctx.Request.ContentLength, maxSize)
			}
		}

		body, err = io.ReadAll(http.MaxBytesReader(ctx.Writer, ctx.Request.Body, ctx.getMaxBodySize()))
		if err != nil {
			return
		}
		ctx.Set(BodyBytesKey, body)
	}

	return
}

// getMaxBodySize returns the maximum body size, defaulting to 10MB
func (ctx *Context) getMaxBodySize() int64 {
	if maxSize, ok := ctx.Get("max_body_size"); ok {
		if size, ok := maxSize.(int64); ok {
			return size
		}
	}
	return DefaultMaxRequestBodySize
}

// SetMaxBodySize sets the maximum request body size for this context
func (ctx *Context) SetMaxBodySize(size int64) {
	ctx.Set("max_body_size", size)
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

// @TODO: cache
func (ctx *Context) QueryParam(name string, defaultValue ...string) string {
	if val := ctx.Request.URL.Query().Get(name); val != "" {
		return val
	}
	for _, v := range defaultValue {
		return v
	}
	return ""
}

func (ctx *Context) QueryParamInt(name string, defaultValue ...int) int {
	str := ctx.QueryParam(name, "0")
	if str == "" {
		for _, v := range defaultValue {
			return v
		}
		return 0
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return val
}

// Host host as string
func (ctx *Context) Host() string {
	return ctx.Request.Host
}

// Host ip as string
func (ctx *Context) Ip() string {
	return ctx.Request.RemoteAddr
}

// Method specifies the HTTP method (GET, POST, PUT, etc.).
func (ctx *Context) Method() string {
	return ctx.Request.Method
}

// UserAgent returns the client's User-Agent, if sent in the request.
func (ctx *Context) UserAgent() string {
	return ctx.Request.UserAgent()
}

// URL request url
func (ctx *Context) URL() *url.URL {
	return ctx.Request.URL
}

// GetContentType returns the Content-Type header of the request.
func (ctx *Context) GetContentType() string {
	return filterFlags(ctx.Request.Header.Get("Content-Type"))
}

// GetCookie returns the named cookie provided in the request or nil if not found.
// If multiple cookies match the given name, only one cookie will be returned.
func (ctx *Context) GetCookie(name string) *http.Cookie {
	// @todo: ctx.Request.readCookies is slow
	if cookie, err := ctx.Request.Cookie(name); err == nil {
		return cookie
	}
	return nil
}

// GetHeader gets the first value associated with the given key from the request headers.
// If there are no values associated with the key, GetHeader returns "".
// It is case-insensitive; http.CanonicalHeaderKey is used to canonicalize the provided key.
//
// Example:
//
//	contentType := ctx.GetHeader("Content-Type")
//	// Returns: "application/json"
func (ctx *Context) GetHeader(key string) string {
	return ctx.Request.Header.Get(key)
}

// GetHeaderValidated gets a header value with validation for size and format
func (ctx *Context) GetHeaderValidated(key string, maxLength int) (string, error) {
	value := ctx.GetHeader(key)
	if err := ValidateHeaderValue(key, value, maxLength); err != nil {
		return "", err
	}
	return value, nil
}

// QueryParamValidated returns a validated query parameter with length checking
func (ctx *Context) QueryParamValidated(name string, maxLength int) (string, error) {
	value := ctx.QueryParam(name)
	if err := ValidateQueryParameter(name, value, maxLength); err != nil {
		return "", err
	}
	return value, nil
}

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}
