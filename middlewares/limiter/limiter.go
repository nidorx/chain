package limiter

import (
	"fmt"
	"net/http"

	"github.com/nidorx/chain"
)

const (
	// DefaultMaxBodySize is the default maximum request body size (10MB).
	DefaultMaxBodySize = 10 << 20 // 10MB
)

// Config holds the configuration for the request size limiter middleware.
type Config struct {
	// MaxBodySize is the maximum allowed request body size in bytes.
	// Default: 10MB (10 << 20)
	MaxBodySize int64

	// ErrorHandler is an optional custom error handler called when the body size exceeds the limit.
	// If not set, returns a default 413 Payload Too Large response.
	ErrorHandler func(ctx *chain.Context)

	// SkipMethods is a list of HTTP methods that should skip the size limit check.
	// Useful for GET, HEAD, OPTIONS which typically don't have bodies.
	// Default: ["GET", "HEAD", "OPTIONS"]
	SkipMethods []string

	// StatusCode is the HTTP status code to return when the body size exceeds the limit.
	// Default: 413 (Payload Too Large)
	StatusCode int
}

// DefaultConfig returns a default configuration for the request size limiter middleware.
func DefaultConfig() Config {
	return Config{
		MaxBodySize: DefaultMaxBodySize,
		SkipMethods: []string{http.MethodGet, http.MethodHead, http.MethodOptions},
		StatusCode:  http.StatusRequestEntityTooLarge,
	}
}

// New creates a request size limiter middleware with the given configuration.
//
// Example:
//
//	// Default 10MB limit
//	router.Use(limiter.New())
//
//	// 1MB limit
//	router.Use(limiter.New(limiter.Config{
//	    MaxBodySize: 1 << 20,
//	}))
//
//	// 5MB limit with custom error
//	router.Use(limiter.New(limiter.Config{
//	    MaxBodySize: 5 << 20,
//	    ErrorHandler: func(ctx *chain.Context) {
//	        ctx.Status(413, map[string]string{"error": "file too large"})
//	    },
//	}))
func New(config ...Config) chain.MiddlewareFunc {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultConfig()
	}

	if cfg.MaxBodySize == 0 {
		cfg.MaxBodySize = DefaultMaxBodySize
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusRequestEntityTooLarge
	}
	if cfg.SkipMethods == nil {
		cfg.SkipMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	}

	// Build skip methods map for fast lookup
	skipMethods := make(map[string]bool, len(cfg.SkipMethods))
	for _, method := range cfg.SkipMethods {
		skipMethods[method] = true
	}

	return func(ctx *chain.Context, next func() error) error {
		// Skip limit for methods that typically don't have bodies
		if skipMethods[ctx.Method()] {
			return next()
		}

		// Wrap the request body with MaxBytesReader
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, cfg.MaxBodySize)

		// Set max body size in context for BodyBytes() to respect
		ctx.SetMaxBodySize(cfg.MaxBodySize)

		err := next()

		// Check if error is due to body size limit
		if err != nil {
			// Check for MaxBytesError or io.EOF with limit exceeded
			if isBodySizeError(err) {
				if ctx.WriteStarted() {
					return err
				}
				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(ctx)
				} else {
					ctx.Error(
						fmt.Sprintf("Request Entity Too Large"),
						cfg.StatusCode,
						map[string]any{
							"error":    "request body too large",
							"max_size": cfg.MaxBodySize,
						},
					)
				}
				return nil
			}
		}

		return err
	}
}

// isBodySizeError checks if an error is related to body size limit.
func isBodySizeError(err error) bool {
	if err == nil {
		return false
	}
	// Check for MaxBytesError
	if _, ok := err.(*http.MaxBytesError); ok {
		return true
	}
	// Check error message for indication of size limit
	return err.Error() == "http: request body too large"
}

// MaxBytes creates a request size limiter middleware with the given maximum body size.
// This is a convenience function for the common case.
//
// Example:
//
//	router.Use(limiter.MaxBytes(1 << 20)) // 1MB
func MaxBytes(maxSize int64) chain.MiddlewareFunc {
	return New(Config{
		MaxBodySize: maxSize,
	})
}
