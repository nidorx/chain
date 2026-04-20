# Timeout Middleware

Request timeout middleware for Chain with proper context cancellation.

## Overview

The timeout middleware enforces request deadlines using Go's `context.Context` cancellation mechanism. When a timeout occurs:

1. The request context is cancelled
2. All code respecting context cancellation stops execution
3. Database transactions roll back
4. HTTP client requests are cancelled
5. A 503 Service Unavailable response is sent (if response not yet written)

This approach ensures that timeouts properly propagate to all downstream code, preventing resource leaks and ensuring clean cancellation.

## Features

- **Context cancellation** — Uses Go's standard `context.WithTimeout` mechanism
- **Database transaction rollback** — Transactions using `QueryContext` automatically cancel
- **HTTP client cancellation** — HTTP requests using context are cancelled
- **Configurable timeout** — Per-route or global timeout settings
- **Custom error responses** — Custom error handlers for timeout responses
- **Path-scoped timeouts** — Apply timeouts to specific route patterns
- **Non-cooperative handler support** — Blocks response even if handler ignores context

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/nidorx/chain"
    "github.com/nidorx/chain/middlewares/timeout"
    "net/http"
    "time"
)

func main() {
    router := chain.New()

    // Apply 30-second timeout to all routes
    router.Use(timeout.New(timeout.Config{
        Timeout: 30 * time.Second,
    }))

    router.GET("/", func(ctx *chain.Context) error {
        ctx.Json(map[string]string{"message": "Hello World"})
        return nil
    })

    http.ListenAndServe(":8080", router)
}
```

## Configuration

### Config Struct

```go
type Config struct {
    // Timeout is the maximum duration for the request
    Timeout time.Duration

    // StatusCode is the HTTP status code to return on timeout
    // Default: 503 (Service Unavailable)
    StatusCode int

    // ErrorHandler is an optional custom error handler called on timeout
    ErrorHandler func(ctx *chain.Context)

    // IncludeTimeoutHeader if true, sets X-Timeout-Seconds header in response
    IncludeTimeoutHeader bool
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Timeout` | `time.Duration` | Required | Maximum request duration |
| `StatusCode` | `int` | `503` | HTTP status on timeout |
| `ErrorHandler` | `func(*chain.Context)` | `nil` | Custom timeout response |
| `IncludeTimeoutHeader` | `bool` | `false` | Add timeout header |

## Usage Examples

### Global Timeout

```go
// Apply 30-second timeout to all routes
router.Use(timeout.New(timeout.Config{
    Timeout: 30 * time.Second,
}))
```

### Path-Scoped Timeout

```go
// Apply 10-second timeout to API routes only
router.Use("/api/*", timeout.New(timeout.Config{
    Timeout: 10 * time.Second,
}))

// Other routes have no timeout
router.GET("/health", healthHandler)
```

### Multiple Timeouts

```go
// Short timeout for public API
router.Use("/api/v1/*", timeout.New(timeout.Config{
    Timeout: 5 * time.Second,
}))

// Longer timeout for admin API
router.Use("/admin/*", timeout.New(timeout.Config{
    Timeout: 60 * time.Second,
}))
```

### Custom Error Response

```go
router.Use(timeout.New(timeout.Config{
    Timeout: 10 * time.Second,
    ErrorHandler: func(ctx *chain.Context) {
        ctx.Json(map[string]string{
            "error": "Request timed out. Please try again.",
        })
    },
}))
```

### Custom Status Code

```go
router.Use(timeout.New(timeout.Config{
    Timeout:    10 * time.Second,
    StatusCode: http.StatusGatewayTimeout, // 504
}))
```

### With Timeout Header

```go
router.Use(timeout.New(timeout.Config{
    Timeout:              30 * time.Second,
    IncludeTimeoutHeader: true,
}))
```

Response will include:
```
X-Timeout-Seconds: 30s
```

### Convenience Function

```go
// Simple timeout with default configuration
router.Use(timeout.WithTimeout(30 * time.Second))
```

## Handler Cooperation

For timeouts to work properly, handlers **must** respect context cancellation.

### Database Queries (GOOD)

```go
router.GET("/users/:id", func(ctx *chain.Context) error {
    // Database driver respects context cancellation
    row := db.QueryRowContext(ctx.Request.Context(),
        "SELECT id, name, email FROM users WHERE id = ?",
        ctx.GetParam("id"),
    )

    var user User
    if err := row.Scan(&user.ID, &user.Name, &user.Email); err != nil {
        if err == context.Canceled {
            // Timeout occurred
            return err
        }
        return err
    }

    ctx.Json(user)
    return nil
})
```

### HTTP Client Requests (GOOD)

```go
router.GET("/external", func(ctx *chain.Context) error {
    // HTTP client respects context cancellation
    req, _ := http.NewRequestWithContext(
        ctx.Request.Context(),
        "GET",
        "https://api.example.com/data",
        nil,
    )

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        if err == context.Canceled {
            // Timeout occurred
            return err
        }
        return err
    }
    defer resp.Body.Close()

    // Process response
    return nil
})
```

### Long-Running Operations (GOOD)

```go
router.GET("/process", func(ctx *chain.Context) error {
    for _, item := range items {
        select {
        case <-ctx.Request.Context().Done():
            // Context cancelled - stop processing
            return ctx.Request.Context().Err()
        default:
            // Process item
            processItem(item)
        }
    }
    ctx.OK()
    return nil
})
```

### Non-Cooperative Handler (BAD)

```go
// This handler ignores context cancellation
router.GET("/slow", func(ctx *chain.Context) error {
    // This blocks for 10 seconds even after timeout
    time.Sleep(10 * time.Second)
    ctx.OK()
    return nil
})
```

The timeout middleware will still return 503, but the handler continues running in the background, consuming resources.

## Context Methods

The `chain.Context` provides methods to check timeout state:

### Deadline()

Check if a timeout is configured:

```go
if deadline, ok := ctx.Deadline(); ok {
    timeLeft := time.Until(deadline)
    slog.Info("time remaining", "seconds", timeLeft.Seconds())
}
```

### Done()

Listen for cancellation:

```go
select {
case <-ctx.Done():
    // Context cancelled
    return ctx.Err()
case <-time.After(1 * time.Second):
    // Normal operation
}
```

### Err()

Check cancellation error:

```go
if err := ctx.Err(); err != nil {
    if err == context.DeadlineExceeded {
        // Timeout occurred
    }
    if err == context.Canceled {
        // Request cancelled (e.g., client disconnected)
    }
}
```

## Integration Examples

### With Recovery and Logger

```go
router := chain.New()

// 1. Recovery (catch panics)
router.Use(recovery.New())

// 2. Logger (log requests)
router.Use(logger.New())

// 3. Timeout (enforce deadlines)
router.Use(timeout.New(timeout.Config{
    Timeout: 30 * time.Second,
}))

// 4. Routes
router.GET("/api/users", usersHandler)
```

### With Database

```go
router.POST("/users", func(ctx *chain.Context) error {
    // Start transaction - will be cancelled on timeout
    tx, err := db.BeginTx(ctx.Request.Context(), nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Execute query - respects context
    _, err = tx.ExecContext(ctx.Request.Context(),
        "INSERT INTO users (name, email) VALUES (?, ?)",
        name, email,
    )
    if err != nil {
        return err
    }

    // Commit transaction
    return tx.Commit()
})
```

### With External API Calls

```go
router.GET("/weather", func(ctx *chain.Context) error {
    client := &http.Client{
        Timeout: 5 * time.Second, // Also set client timeout
    }

    req, _ := http.NewRequestWithContext(
        ctx.Request.Context(),
        "GET",
        "https://api.weather.com/current",
        nil,
    )

    resp, err := client.Do(req)
    if err != nil {
        if err == context.DeadlineExceeded {
            return ctx.Err()
        }
        return err
    }
    defer resp.Body.Close()

    // Process response
    return nil
})
```

## API Reference

### Functions

```go
// DefaultConfig returns a default configuration (30s timeout)
func DefaultConfig() Config

// New creates timeout middleware with configuration
func New(config ...Config) chain.MiddlewareFunc

// WithTimeout creates timeout middleware with specified duration
func WithTimeout(timeout time.Duration) chain.MiddlewareFunc
```

### Types

```go
type Config struct {
    Timeout              time.Duration
    StatusCode           int
    ErrorHandler         func(ctx *chain.Context)
    IncludeTimeoutHeader bool
}
```

### Error Values

```go
// ErrRequestTimeout is returned when a request exceeds its timeout
var ErrRequestTimeout = errors.New("request timeout")
```

## Best Practices

1. **Always set timeouts** — Prevents resource exhaustion from hung requests
2. **Use shorter timeouts for internal services** — 5-10s for APIs
3. **Use longer timeouts for complex operations** — 30-60s for batch processing
4. **Handlers should check `ctx.Done()`** — Enables cooperative cancellation
5. **Database queries should use `QueryContext`** — Automatic rollback on timeout
6. **HTTP clients should use context** — Cancels in-flight requests
7. **Set both middleware and client timeouts** — Defense in depth

## Performance Considerations

### Goroutine Usage

The timeout middleware spawns one goroutine per request to monitor timeout. This is a standard Go pattern and has minimal overhead:

- Goroutine creation: ~2μs
- Goroutine memory: ~2KB initial, grows as needed
- Context cancellation: O(1)

### Non-Cooperative Handlers

If handlers don't respect context cancellation:

- Handler continues executing
- Memory consumed until handler completes
- Response blocked by timeout middleware
- Eventually handler completes and resources released

For best results, ensure all handlers check `ctx.Done()`.

## Troubleshooting

### Timeout Not Triggering

**Cause:** Handler completes before timeout.

**Solution:** Reduce timeout duration or check handler execution time.

```go
router.Use(timeout.New(timeout.Config{
    Timeout: 1 * time.Second, // Shorter timeout
}))
```

### Handler Continues After Timeout

**Cause:** Handler doesn't respect context cancellation.

**Solution:** Update handler to check `ctx.Done()`:

```go
select {
case <-ctx.Request.Context().Done():
    return ctx.Request.Context().Err()
default:
    // Continue processing
}
```

### Database Transaction Not Rolling Back

**Cause:** Transaction not using context-aware methods.

**Solution:** Use `BeginTx` and `QueryContext` with request context:

```go
tx, err := db.BeginTx(ctx.Request.Context(), nil)
rows, err := tx.QueryContext(ctx.Request.Context(), "SELECT ...")
```

## See Also

- [context package documentation](https://pkg.go.dev/context)
- [Chain Middleware Overview](../README.md)
- [Chain Router Documentation](../../README.md)
