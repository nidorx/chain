# Recovery Middleware

Panic recovery middleware for Chain with stack trace support.

## Overview

The recovery middleware catches panics during request processing and handles them gracefully. It prevents your server from crashing due to unhandled panics in middleware or handlers, logs the panic details with stack traces, and returns a proper HTTP error response to the client.

## Features

- **Automatic recovery** — Catches panics and recovers gracefully
- **Stack trace logging** — Detailed stack traces for debugging
- **Custom handlers** — Provide custom recovery logic
- **Configurable output** — Control what gets logged
- **Stack buffer size** — Adjust stack trace buffer size
- **Panic logging toggle** — Enable/disable panic logging

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/nidorx/chain"
    "github.com/nidorx/chain/middlewares/recovery"
    "net/http"
)

func main() {
    router := chain.New()
    
    // Add recovery as FIRST middleware
    router.Use(recovery.New())
    
    router.GET("/", func(ctx *chain.Context) error {
        ctx.Json(map[string]string{"message": "Hello World"})
        return nil
    })
    
    http.ListenAndServe(":8080", router)
}
```

**Important:** Always register recovery as the **first** middleware to catch panics from all subsequent middleware and handlers.

## Configuration

### Config Struct

```go
type Config struct {
    // Custom logger instance (default: slog.Default())
    Logger *slog.Logger
    
    // Print stack trace when panic occurs (default: true)
    PrintStack bool
    
    // Stack buffer size in bytes (default: 4096)
    StackSize int
    
    // Custom recovery handler function
    RecoveryHandler func(ctx *chain.Context, err any)
    
    // Disable panic logging (default: false)
    DisablePanicLogging bool
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Logger` | `*slog.Logger` | `slog.Default()` | Logger instance |
| `PrintStack` | `bool` | `true` | Print stack trace |
| `StackSize` | `int` | `4096` | Stack buffer size |
| `RecoveryHandler` | `func(*chain.Context, any)` | `nil` | Custom recovery function |
| `DisablePanicLogging` | `bool` | `false` | Disable panic logging |

## Usage Examples

### Default Recovery

```go
router.Use(recovery.New())
```

When a panic occurs:
1. Logs the panic with stack trace
2. Returns 500 Internal Server Error
3. Server continues running

### Custom Recovery Handler

```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        // Log to external service
        reportToSentry(err)
        
        // Return custom error response
        ctx.Json(map[string]any{
            "error":   "internal server error",
            "message": "something went wrong",
        })
    },
}))
```

### Disable Stack Trace

```go
router.Use(recovery.New(recovery.Config{
    PrintStack: false,
}))
```

Logs panic without stack trace (useful in production to reduce log volume).

### Custom Stack Size

```go
router.Use(recovery.New(recovery.Config{
    StackSize: 8192, // 8KB stack buffer
}))
```

Increase stack size if you're seeing truncated stack traces.

### Disable Panic Logging

```go
router.Use(recovery.New(recovery.Config{
    DisablePanicLogging: true,
}))
```

Recovers from panics but doesn't log them. Use when you have custom logging in `RecoveryHandler`.

### Custom Logger

```go
import (
    "log/slog"
    "os"
)

// Create error-only logger
errorLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelError,
}))

router.Use(recovery.New(recovery.Config{
    Logger: errorLogger,
}))
```

## Integration Examples

### With Sentry/Error Tracking

```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        // Report to Sentry
        sentry.WithScope(func(scope *sentry.Scope) {
            scope.SetTag("path", ctx.URL().Path)
            scope.SetTag("method", ctx.Method())
            sentry.CurrentHub().Recover(err)
        }),
        
        // Return user-friendly error
        ctx.Json(map[string]string{
            "error": "internal server error",
        })
    },
}))
```

### With Custom JSON Error Format

```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        ctx.Status(500, map[string]any{
            "success": false,
            "error": map[string]string{
                "code":    "INTERNAL_ERROR",
                "message": "An unexpected error occurred",
            },
        })
    },
}))
```

### With Logging Middleware

```go
router := chain.New()

// Recovery catches panics
router.Use(recovery.New())

// Logger logs all requests
router.Use(logger.New())

// Routes
router.GET("/panic", func(ctx *chain.Context) error {
    panic("test panic")
})
```

When `/panic` is requested:
1. Panic occurs in handler
2. Recovery middleware catches it
3. Recovery logs the panic
4. Logger middleware logs the request (with 500 status)
5. Client receives 500 error

### In Production

```go
// Create production logger
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelError,
})
prodLogger := slog.New(handler)

router.Use(recovery.New(recovery.Config{
    Logger:     prodLogger,
    PrintStack: false, // Don't print stacks in production
}))
```

### In Development

```go
router.Use(recovery.New(recovery.Config{
    PrintStack: true,    // Show full stack traces
    Logger:     nil,     // Use default logger
}))
```

## Best Practices

### Always Use Recovery First

```go
router := chain.New()

// ✅ GOOD - Recovery is first
router.Use(recovery.New())
router.Use(logger.New())
router.Use(cors.Default())

// ❌ BAD - Recovery is not first
router.Use(logger.New())
router.Use(recovery.New()) // Won't catch logger panics
```

### Use Custom Handler in Production

```go
router.Use(recovery.New(recovery.Config{
    PrintStack: false, // Reduce log noise
    RecoveryHandler: func(ctx *chain.Context, err any) {
        // Log to monitoring service
        logError(err)
        
        // Return generic error
        ctx.Error("Internal Server Error", 500)
    },
}))
```

### Include Request Context

```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        slog.Error(
            "panic recovered",
            "error", err,
            "method", ctx.Method(),
            "path", ctx.URL().Path,
            "ip", ctx.Ip(),
            "user_agent", ctx.UserAgent(),
        )
    },
}))
```

## API Reference

### Functions

```go
// DefaultConfig returns a default configuration
func DefaultConfig() Config

// New creates recovery middleware
func New(config ...Config) chain.MiddlewareFunc

// ForceStackPanic helper function for testing
func ForceStackPanic(message string)
```

### Types

```go
type Config struct {
    Logger            *slog.Logger
    PrintStack        bool
    StackSize         int
    RecoveryHandler   func(ctx *chain.Context, err any)
    DisablePanicLogging bool
}
```

## Panic Recovery Flow

```
Request arrives
    │
    ▼
Recovery middleware
    │
    ├─ defer func() {
    │     if rcv := recover(); rcv != nil {
    │         // 1. Log panic with stack
    │         // 2. Call RecoveryHandler (if set)
    │         // 3. Return 500 error (default)
    │     }
    │   }()
    │
    ▼
Next middleware/handler
    │
    ├─ Executes normally
    │   └─ Returns error or success
    │
    └─ Panics
        │
        ▼
    defer recovers
        │
        ▼
    Error response sent
```

## Stack Traces

### Example Stack Trace

```
goroutine 1 [running]:
runtime/debug.Stack()
    /usr/local/go/src/runtime/debug/stack.go:24 +0x5e
github.com/nidorx/chain/middlewares/recovery.stack(0x1000)
    /path/to/recovery.go:123 +0x26
github.com/nidorx/chain/middlewares/recovery.New.func1.1()
    /path/to/recovery.go:78 +0x9f
panic({0x1234567, 0x12345678})
    /usr/local/go/src/runtime/panic.go:914 +0x21f
main.main.func1(0xc000123456)
    /path/to/main.go:45 +0x3e
```

### Stack Buffer Size

The `StackSize` config option controls the buffer size for stack traces:

- **Default:** 4096 bytes (4KB)
- **Increase to:** 8192 bytes if traces are truncated
- **Decrease to:** 2048 bytes to reduce memory usage

```go
router.Use(recovery.New(recovery.Config{
    StackSize: 8192,
}))
```

## Error Handling

### Default Behavior

When a panic is recovered:

1. Panic is logged with stack trace (if `PrintStack: true`)
2. If response hasn't been written:
   - Returns 500 Internal Server Error
   - Error message: `"500 Internal Server Error: <panic message>"`
3. If response has been written:
   - No additional response is sent
   - Panic is still logged

### Custom Error Response

```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        if !ctx.WriteStarted() {
            ctx.Status(500, map[string]any{
                "success": false,
                "error": map[string]string{
                    "code":    "INTERNAL_ERROR",
                    "message": "Internal server error",
                },
            })
        }
    },
}))
```

## Testing

### Test Recovery

```go
func TestRecovery(t *testing.T) {
    router := chain.New()
    router.Use(recovery.New())
    
    router.GET("/panic", func(ctx *chain.Context) error {
        panic("test panic")
    })
    
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/panic", nil)
    router.ServeHTTP(w, req)
    
    if w.Code != 500 {
        t.Errorf("Expected 500, got %d", w.Code)
    }
}
```

### Test Custom Handler

```go
func TestRecoveryCustomHandler(t *testing.T) {
    handlerCalled := false
    
    router := chain.New()
    router.Use(recovery.New(recovery.Config{
        RecoveryHandler: func(ctx *chain.Context, err any) {
            handlerCalled = true
        },
    }))
    
    router.GET("/panic", func(ctx *chain.Context) error {
        panic("test panic")
    })
    
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/panic", nil)
    router.ServeHTTP(w, req)
    
    if !handlerCalled {
        t.Error("Expected recovery handler to be called")
    }
}
```

### Force Panic for Testing

```go
func TestForcePanic(t *testing.T) {
    defer func() {
        if rcv := recover(); rcv == nil {
            t.Error("Expected panic")
        }
    }()
    
    recovery.ForceStackPanic("test panic")
}
```

## Troubleshooting

### Panic Not Recovered

**Cause:** Recovery middleware not registered or registered after panicking middleware.

**Solution:**
```go
// ✅ GOOD - Recovery is first
router.Use(recovery.New())
router.Use(otherMiddleware)
```

### Stack Trace Truncated

**Cause:** Stack buffer size too small.

**Solution:**
```go
router.Use(recovery.New(recovery.Config{
    StackSize: 8192,
}))
```

### No Error Response Sent

**Cause:** Response was already written before panic.

**Solution:**
```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        if ctx.WriteStarted() {
            slog.Warn("panic recovered but response already sent", "error", err)
            return
        }
        ctx.Error("Internal Server Error", 500)
    },
}))
```

### Excessive Log Volume

**Cause:** Stack traces printed for every panic in production.

**Solution:**
```go
router.Use(recovery.New(recovery.Config{
    PrintStack: false,
}))
```

## Security Considerations

### Don't Expose Stack Traces to Clients

```go
// ❌ BAD - Exposes internals
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        ctx.Error(fmt.Sprintf("Error: %v\nStack: %s", err, getStack()), 500)
    },
}))

// ✅ GOOD - Generic error message
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        ctx.Error("Internal Server Error", 500)
    },
}))
```

### Log Panics Securely

Avoid logging sensitive information in stack traces:

```go
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        // Log panic without sensitive data
        slog.Error(
            "panic recovered",
            "error", err,
            "path", ctx.URL().Path,
            // Don't log request body or headers
        )
    },
}))
```

## See Also

- [Go Panic Handling](https://go.dev/doc/effective_go#errors)
- [Chain Middleware Overview](../README.md)
- [Logger Middleware](../logger/README.md)
