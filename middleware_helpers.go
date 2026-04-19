package chain

import "net/http"

// MiddlewareFunc is the standard middleware function signature
type MiddlewareFunc func(ctx *Context, next func() error) error

// MiddlewareChain is a helper type that manages a chain of middleware functions
type MiddlewareChain struct {
	middlewares []MiddlewareFunc
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain(middlewares ...MiddlewareFunc) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

// Add appends middleware to the chain
func (mc *MiddlewareChain) Add(middleware MiddlewareFunc) {
	mc.middlewares = append(mc.middlewares, middleware)
}

// Execute runs the middleware chain with the given handler
func (mc *MiddlewareChain) Execute(ctx *Context, handler func() error) error {
	var exec func(int) func() error
	exec = func(index int) func() error {
		if index >= len(mc.middlewares) {
			return handler
		}
		return func() error {
			return mc.middlewares[index](ctx, exec(index+1))
		}
	}
	return exec(0)()
}

// WrapHandler wraps an http.Handler with middleware chain
func WrapHandler(handler http.Handler) MiddlewareFunc {
	return func(ctx *Context, next func() error) error {
		spy := &ResponseWriterSpy{ResponseWriter: ctx.Writer}
		handler.ServeHTTP(spy, ctx.Request)
		if spy.writeStarted {
			return nil
		}
		return next()
	}
}

// MaxBytesMiddleware creates middleware that limits request body size
func MaxBytesMiddleware(maxSize int64) MiddlewareFunc {
	return func(ctx *Context, next func() error) error {
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxSize)
		return next()
	}
}

// ContentTypeMiddleware creates middleware that validates Content-Type header
func ContentTypeMiddleware(allowedTypes ...string) MiddlewareFunc {
	return func(ctx *Context, next func() error) error {
		contentType := ctx.GetContentType()
		if contentType == "" {
			return next()
		}

		for _, allowed := range allowedTypes {
			if contentType == allowed || contentTypeStartsWith(contentType, allowed) {
				return next()
			}
		}

		ctx.Json(map[string]string{
			"error": "Unsupported Media Type",
		})
		ctx.Status(http.StatusUnsupportedMediaType)
		return nil
	}
}

func contentTypeStartsWith(contentType string, prefix string) bool {
	if len(contentType) < len(prefix) {
		return false
	}
	return contentType[:len(prefix)] == prefix
}
