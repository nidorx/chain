// Package timeout provides request timeout middleware for Chain.
//
// The timeout middleware enforces request deadlines using Go's context
// cancellation mechanism. When a timeout occurs:
//   - The request context is cancelled
//   - All code respecting context cancellation stops execution
//   - Database transactions roll back
//   - HTTP client requests are cancelled
//   - A 503 Service Unavailable response is sent (if response not yet written)
//
// Basic usage:
//
//	router.Use(timeout.New(timeout.Config{
//	    Timeout: 30 * time.Second,
//	}))
//
// With path-scoped timeout:
//
//	router.Use("/api/*", timeout.New(timeout.Config{
//	    Timeout: 10 * time.Second,
//	}))
//
// Handlers should respect context cancellation for proper timeout enforcement:
//
//	router.GET("/db", func(ctx *chain.Context) error {
//	    // Database driver will respect context cancellation
//	    rows, err := db.QueryContext(ctx.Request.Context(), "SELECT ...")
//	    if err != nil {
//	        return err // Will be context.Canceled if timeout
//	    }
//	    return nil
//	})
package timeout

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nidorx/chain"
)

// ErrRequestTimeout is returned when a request exceeds its timeout duration.
var ErrRequestTimeout = errors.New("request timeout")

// timeoutResponseWriter wraps an http.ResponseWriter and blocks writes after timeout.
type timeoutResponseWriter struct {
	http.ResponseWriter
	timedOut     *atomic.Bool
	writeStarted *atomic.Bool
}

func (w *timeoutResponseWriter) WriteHeader(code int) {
	if w.timedOut.Load() {
		return // Don't write after timeout
	}
	w.writeStarted.Store(true)
	w.ResponseWriter.WriteHeader(code)
}

func (w *timeoutResponseWriter) Write(b []byte) (int, error) {
	if w.timedOut.Load() {
		return 0, http.ErrHandlerTimeout
	}
	w.writeStarted.Store(true)
	return w.ResponseWriter.Write(b)
}

func (w *timeoutResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Config holds the configuration for the timeout middleware.
type Config struct {
	// Timeout is the maximum duration for the request.
	// Required. If zero or negative, the middleware is a no-op.
	Timeout time.Duration

	// StatusCode is the HTTP status code to return on timeout.
	// Default: 503 (Service Unavailable)
	StatusCode int

	// ErrorHandler is an optional custom error handler called on timeout.
	// If not set, returns a default 503 response with JSON error.
	ErrorHandler func(ctx *chain.Context)

	// IncludeTimeoutHeader if true, sets X-Timeout-Seconds header in response.
	// Default: false
	IncludeTimeoutHeader bool
}

// DefaultConfig returns a default configuration for the timeout middleware.
func DefaultConfig() Config {
	return Config{
		Timeout:              30 * time.Second,
		StatusCode:           http.StatusServiceUnavailable,
		IncludeTimeoutHeader: false,
	}
}

// New creates a timeout middleware with the given configuration.
//
// Example:
//
//	// Default 30-second timeout
//	router.Use(timeout.New())
//
//	// Custom timeout
//	router.Use(timeout.New(timeout.Config{
//	    Timeout: 10 * time.Second,
//	}))
//
//	// With custom error handler
//	router.Use(timeout.New(timeout.Config{
//	    Timeout: 5 * time.Second,
//	    ErrorHandler: func(ctx *chain.Context) {
//	        ctx.Json(map[string]string{"error": "request timed out"})
//	    },
//	}))
func New(config ...Config) chain.MiddlewareFunc {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultConfig()
	}

	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusServiceUnavailable
	}

	return func(ctx *chain.Context, next func() error) error {
		if cfg.Timeout <= 0 {
			return next()
		}

		if cfg.IncludeTimeoutHeader {
			ctx.SetHeader("X-Timeout-Seconds", cfg.Timeout.String())
		}

		var (
			mu         sync.Mutex
			timedOut   bool
			written    bool
			origWriter = ctx.Writer
		)

		// Wrapper to block writes after timeout
		wrapper := &writeBlocker{
			ResponseWriter: ctx.Writer,
			checkWrite: func() bool {
				mu.Lock()
				defer mu.Unlock()
				if timedOut {
					return false
				}
				written = true
				return true
			},
		}

		// Create a new context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx.Request.Context(), cfg.Timeout)
		defer cancel()

		// Replace the request context so downstream code respects the timeout
		ctx.Request = ctx.Request.WithContext(timeoutCtx)

		// Replace the writer BEFORE starting the goroutine
		ctx.Writer = wrapper

		// Execute the handler chain in a goroutine so we can detect timeout
		errChan := make(chan error, 1)
		go func() {
			errChan <- next()
		}()

		// Wait for either completion or timeout
		select {
		case err := <-errChan:
			return err
		case <-timeoutCtx.Done():
			// Timeout occurred
			mu.Lock()
			timedOut = true
			alreadyWritten := written
			mu.Unlock()

			if !alreadyWritten {
				// Restore original writer for custom error handler
				ctx.Writer = origWriter
				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(ctx)
				} else {
					ctx.Writer.WriteHeader(cfg.StatusCode)
				}
			}

			// Wait for handler to complete (don't return yet)
			// This ensures the handler's late writes are blocked
			<-errChan
			return ErrRequestTimeout
		}
	}
}

// writeBlocker is an http.ResponseWriter wrapper that can be told to block writes.
type writeBlocker struct {
	http.ResponseWriter
	checkWrite func() bool // returns false if write should be blocked
}

func (w *writeBlocker) WriteHeader(code int) {
	if w.checkWrite() {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *writeBlocker) Write(b []byte) (int, error) {
	if !w.checkWrite() {
		return 0, http.ErrHandlerTimeout
	}
	return w.ResponseWriter.Write(b)
}

func (w *writeBlocker) Header() http.Header {
	return w.ResponseWriter.Header()
}

// WithTimeout creates a timeout middleware with the specified duration.
// This is a convenience function for the common case.
//
// Example:
//
//	router.Use(timeout.WithTimeout(30 * time.Second))
func WithTimeout(timeout time.Duration) chain.MiddlewareFunc {
	return New(Config{
		Timeout: timeout,
	})
}
