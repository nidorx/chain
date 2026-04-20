// Package recovery provides panic recovery middleware for Chain.
//
// The recovery middleware catches panics during request processing and:
// - Recovers from the panic gracefully
// - Logs the panic with stack trace
// - Returns a 500 Internal Server Error response
// - Optionally calls a custom recovery handler
//
// Basic usage:
//
//	router.Use(recovery.New())
//
// With custom configuration:
//
//	router.Use(recovery.New(recovery.Config{
//	    PrintStack: true,
//	    RecoveryHandler: func(ctx *chain.Context, err any) {
//	        // Custom recovery logic
//	    },
//	}))
//
// It's recommended to use recovery as the first middleware in the chain
// to catch panics from all subsequent middlewares and handlers.
package recovery

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"github.com/nidorx/chain"
)

// Config holds the configuration for the recovery middleware.
type Config struct {
	// Logger is the slog logger instance. If nil, uses slog.Default().
	Logger *slog.Logger

	// PrintStack controls whether to print the stack trace when a panic occurs.
	// Default: true
	PrintStack bool

	// StackSize is the size of the stack buffer to allocate when printing stack traces.
	// Default: 4096 bytes
	StackSize int

	// RecoveryHandler is an optional custom handler called when a panic is recovered.
	// It receives the context and the panic value.
	// If set, this handler is called instead of returning 500 Internal Server Error.
	RecoveryHandler func(ctx *chain.Context, err any)

	// DisablePanicLogging, if true, prevents logging panics.
	// Default: false
	DisablePanicLogging bool
}

// DefaultConfig returns a default configuration for the recovery middleware.
func DefaultConfig() Config {
	return Config{
		Logger:     slog.Default(),
		PrintStack: true,
		StackSize:  4096,
	}
}

// New creates a recovery middleware with the given configuration.
//
// Example:
//
//	// Default recovery
//	router.Use(recovery.New())
//
//	// With custom handler
//	router.Use(recovery.New(recovery.Config{
//	    RecoveryHandler: func(ctx *chain.Context, err any) {
//	        ctx.Json(map[string]string{"error": "internal server error"})
//	    },
//	}))
func New(config ...Config) chain.MiddlewareFunc {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultConfig()
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.StackSize == 0 {
		cfg.StackSize = 4096
	}
	if !cfg.PrintStack {
		cfg.PrintStack = false
	}

	return func(ctx *chain.Context, next func() error) error {
		defer func() {
			if rcv := recover(); rcv != nil {
				// Log the panic
				if !cfg.DisablePanicLogging {
					if cfg.PrintStack {
						stack := stack(cfg.StackSize)
						cfg.Logger.Error(
							"panic recovered",
							slog.Any("error", rcv),
							slog.String("stack", stack),
							slog.String("path", ctx.URL().Path),
							slog.String("method", ctx.Method()),
						)
					} else {
						cfg.Logger.Error(
							"panic recovered",
							slog.Any("error", rcv),
							slog.String("path", ctx.URL().Path),
							slog.String("method", ctx.Method()),
						)
					}
				}

				// Call custom recovery handler if set
				if cfg.RecoveryHandler != nil {
					cfg.RecoveryHandler(ctx, rcv)
					return
				}

				// Default: return 500 if response hasn't been written
				if !ctx.WriteStarted() {
					ctx.Error(
						fmt.Sprintf("500 Internal Server Error: %v", rcv),
						http.StatusInternalServerError,
					)
				}
			}
		}()

		return next()
	}
}

// stack returns a stack trace string of the given size.
func stack(size int) string {
	buf := make([]byte, size)
	n := runtime.Stack(buf, false)
	buf = buf[:n]

	// Clean up the stack trace
	lines := strings.Split(string(buf), "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// ForceStackPanic is a helper function to test recovery middleware.
// It will always panic with the given message.
//
// This is useful for testing recovery middleware configuration.
//
// Example:
//
//	// In tests
//	func TestRecovery(t *testing.T) {
//	    router := chain.New()
//	    router.Use(recovery.New())
//	    router.GET("/panic", func(ctx *chain.Context) {
//	        recovery.ForceStackPanic("test panic")
//	    })
//	    // ... test that panic was recovered
//	}
func ForceStackPanic(message string) {
	panic(message)
}
