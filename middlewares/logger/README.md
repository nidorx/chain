# Logger Middleware

Structured logging middleware for Chain using Go's `log/slog` package.

## Overview

The logger middleware provides comprehensive request logging with support for multiple formats, request ID tracking, and configurable log levels. It helps you monitor your application's behavior, track request latency, and debug issues in production.

## Features

- **Structured logging** — Uses Go's standard `log/slog` package
- **Request tracking** — Automatic request ID generation
- **Latency measurement** — Track request duration
- **Status code logging** — Log HTTP response status
- **Multiple formats** — Default, combined, JSON, custom
- **Configurable levels** — Different log levels for different status codes
- **Skip paths** — Exclude health checks and static assets
- **Latency warnings** — Alert on slow requests
- **Request ID support** — Generate or pass through request IDs

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/nidorx/chain"
    "github.com/nidorx/chain/middlewares/logger"
    "net/http"
)

func main() {
    router := chain.New()
    
    // Add logging (recommended: after recovery middleware)
    router.Use(logger.New())
    
    router.GET("/", func(ctx *chain.Context) error {
        ctx.Json(map[string]string{"message": "Hello World"})
        return nil
    })
    
    http.ListenAndServe(":8080", router)
}
```

**Output:**
```
2024/04/19 10:30:45 INFO [200] GET /api/users — 12.5ms
2024/04/19 10:30:46 INFO [201] POST /api/users — 45.2ms
2024/04/19 10:30:47 WARN [404] GET /api/missing — 2.1ms
2024/04/19 10:30:48 ERROR [500] GET /api/error — 105.3ms
```

## Configuration

### Config Struct

```go
type Config struct {
    // Log format (default, combined, json, custom)
    Format Format
    
    // Custom format string (when Format is FormatCustom)
    CustomFormat string
    
    // Custom logger instance (default: slog.Default())
    Logger *slog.Logger
    
    // Paths to skip logging
    SkipPaths []string
    
    // Path prefixes to skip logging
    SkipPathPrefixes []string
    
    // Request ID header name (default: "X-Request-ID")
    RequestIDHeader string
    
    // Generate request ID if not present (default: true)
    GenerateRequestID bool
    
    // Custom status level function
    StatusLevelFunc func(status int) slog.Level
    
    // Latency threshold for warnings
    LatencyThreshold time.Duration
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Format` | `Format` | `FormatDefault` | Log format |
| `CustomFormat` | `string` | `""` | Custom format string |
| `Logger` | `*slog.Logger` | `slog.Default()` | Logger instance |
| `SkipPaths` | `[]string` | `[]` | Paths to skip |
| `SkipPathPrefixes` | `[]string` | `[]` | Path prefixes to skip |
| `RequestIDHeader` | `string` | `"X-Request-ID"` | Request ID header name |
| `GenerateRequestID` | `bool` | `true` | Generate request ID |
| `StatusLevelFunc` | `func(int) slog.Level` | `nil` | Custom log levels |
| `LatencyThreshold` | `time.Duration` | `0` | Slow request warning |

## Usage Examples

### Default Logging

```go
router.Use(logger.New())
```

Output format: `[status] method path — latency`

### JSON Logging

```go
router.Use(logger.New(logger.Config{
    Format: logger.FormatJSON,
}))
```

**Output:**
```json
{"time":"2024-04-19T10:30:45Z","level":"INFO","msg":"HTTP Request","method":"GET","path":"/api/users","status":200,"latency":"12.5ms"}
```

### Combined Log Format (Apache-style)

```go
router.Use(logger.New(logger.Config{
    Format: logger.FormatCombined,
}))
```

**Output:**
```
192.168.1.1 - - [19/Apr/2024:10:30:45 -0700] "GET /api/users HTTP/1.1" 200 0
```

### Custom Format

```go
router.Use(logger.New(logger.Config{
    Format:       logger.FormatCustom,
    CustomFormat: "%{method} %{path} %{status} %{latency}",
}))
```

### Available Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `%{method}` | HTTP method | `GET` |
| `%{path}` | Request path | `/api/users` |
| `%{status}` | Response status | `200` |
| `%{latency}` | Request duration | `12.5ms` |
| `%{ip}` | Client IP | `192.168.1.1` |
| `%{useragent}` | User-Agent header | `Mozilla/5.0` |
| `%{referer}` | Referer header | `https://example.com` |
| `%{host}` | Request host | `api.example.com` |
| `%{proto}` | HTTP protocol | `HTTP/1.1` |
| `%{reqid}` | Request ID | `4r9Dm2nK...` |
| `%{err}` | Error message | `not found` |
| `%{query}` | Query string | `page=1&limit=10` |

### Skip Health Checks

```go
router.Use(logger.New(logger.Config{
    SkipPaths: []string{"/health", "/ping", "/metrics"},
}))
```

### Skip Path Prefixes

```go
router.Use(logger.New(logger.Config{
    SkipPathPrefixes: []string{"/static/", "/assets/", "/favicon.ico"},
}))
```

### Custom Logger

```go
import (
    "log/slog"
    "os"
)

// Create custom logger
customLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

router.Use(logger.New(logger.Config{
    Logger: customLogger,
}))
```

### Request ID from Header

```go
// Client sends: X-Request-ID: abc123
router.Use(logger.New(logger.Config{
    RequestIDHeader: "X-Request-ID",
}))
```

The request ID is stored in the context and can be retrieved in handlers:

```go
router.GET("/api/users", func(ctx *chain.Context) error {
    requestID := logger.GetRequestID(ctx)
    // Use in error messages, logs, etc.
    slog.Info("fetching users", "request_id", requestID)
    return nil
})
```

### Auto-Generate Request ID

```go
router.Use(logger.New(logger.Config{
    GenerateRequestID: true,
}))
```

Generates a KSUID (K-Sortable Unique IDentifier) for each request if no `X-Request-ID` header is present.

### Custom Log Levels

```go
router.Use(logger.New(logger.Config{
    StatusLevelFunc: func(status int) slog.Level {
        switch {
        case status >= 500:
            return slog.LevelError
        case status >= 400:
            return slog.LevelWarn
        case status >= 300:
            return slog.LevelInfo
        default:
            return slog.LevelDebug
        }
    },
}))
```

### Slow Request Warnings

```go
router.Use(logger.New(logger.Config{
    LatencyThreshold: 500 * time.Millisecond,
}))
```

Logs a warning when requests take longer than the threshold:

```
2024/04/19 10:30:45 WARN slow request path=/api/reports latency=1.2s threshold=500ms
```

## Integration Examples

### With Recovery

Always place logger after recovery to catch all requests:

```go
router := chain.New()

// 1. Recovery (catch panics)
router.Use(recovery.New())

// 2. Logger (log all requests)
router.Use(logger.New())

// 3. Routes
router.GET("/api/users", usersHandler)
```

### With CORS

```go
router.Use(recovery.New())
router.Use(logger.New())
router.Use(cors.Default())

router.GET("/api/data", func(ctx *chain.Context) error {
    ctx.Json(data)
    return nil
})
```

### With Custom Format and Skip

```go
router.Use(logger.New(logger.Config{
    Format: logger.FormatCustom,
    CustomFormat: "%{method} %{path} %{status} %{latency} req=%{reqid}",
    SkipPaths: []string{"/health"},
    SkipPathPrefixes: []string{"/static/"},
}))
```

### Production Configuration

```go
// Create production logger
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
prodLogger := slog.New(handler)

router.Use(logger.New(logger.Config{
    Logger:           prodLogger,
    Format:           logger.FormatJSON,
    SkipPaths:        []string{"/health", "/ping"},
    LatencyThreshold: 1 * time.Second,
}))
```

## API Reference

### Functions

```go
// DefaultConfig returns a default configuration
func DefaultConfig() Config

// New creates logging middleware
func New(config ...Config) chain.MiddlewareFunc

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx *chain.Context) string
```

### Types

```go
type Format string

const (
    FormatDefault  Format = "default"
    FormatCombined Format = "combined"
    FormatJSON     Format = "json"
    FormatCustom   Format = "custom"
)

type Config struct {
    Format            Format
    CustomFormat      string
    Logger            *slog.Logger
    SkipPaths         []string
    SkipPathPrefixes  []string
    RequestIDHeader   string
    GenerateRequestID bool
    StatusLevelFunc   func(status int) slog.Level
    LatencyThreshold  time.Duration
}
```

## Log Levels

### Default Behavior

| Status Code | Log Level | Example |
|-------------|-----------|---------|
| 2xx, 3xx | `INFO` | Successful requests |
| 4xx | `WARN` | Client errors (404, 400, etc.) |
| 5xx | `ERROR` | Server errors (500, 502, etc.) |

### Custom Levels

Override with `StatusLevelFunc`:

```go
StatusLevelFunc: func(status int) slog.Level {
    if status >= 400 {
        return slog.LevelError // All errors at ERROR level
    }
    return slog.LevelDebug // Success at DEBUG level
}
```

## Request IDs

### How It Works

1. Middleware checks for `X-Request-ID` header in request
2. If present, uses that value
3. If not present and `GenerateRequestID` is true, generates KSUID
4. Stores in context for retrieval in handlers

### Using Request IDs

```go
// In handler
func handler(ctx *chain.Context) error {
    requestID := logger.GetRequestID(ctx)
    
    slog.Info(
        "processing request",
        "request_id", requestID,
        "user_id", userID,
    )
    
    // Pass to downstream services
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-Request-ID", requestID)
    
    return nil
}
```

### Correlation Logging

Use request IDs to correlate logs across services:

```go
// Service A
router.Use(logger.New())
router.GET("/api/data", func(ctx *chain.Context) error {
    requestID := logger.GetRequestID(ctx)
    
    // Call Service B with same request ID
    req, _ := http.NewRequest("GET", "http://service-b/api", nil)
    req.Header.Set("X-Request-ID", requestID)
    
    // Both services share same request ID in logs
    return nil
})
```

## Performance Considerations

### Logging Overhead

The logger middleware adds minimal overhead:
- Time measurement: ~100ns
- Log writing: ~50μs (depends on handler)
- Request ID generation: ~200ns

### JSON Handler Performance

For high-throughput applications, use a performant JSON handler:

```go
import "github.com/phsym/console-slog"

handler := console.NewHandler(os.Stdout, &console.HandlerOptions{
    Level: slog.LevelInfo,
})
router.Use(logger.New(logger.Config{
    Logger: slog.New(handler),
}))
```

### Skip Unnecessary Logs

Skip health checks and static assets:

```go
router.Use(logger.New(logger.Config{
    SkipPaths:        []string{"/health", "/ping"},
    SkipPathPrefixes: []string{"/static/", "/assets/"},
}))
```

## Structured Logging

### Adding Context

The logger automatically adds:
- `method` — HTTP method
- `path` — Request path
- `status` — Response status code
- `latency` — Request duration
- `request_id` — Request ID (if present)
- `ip` — Client IP (if available)
- `user_agent` — User-Agent header (if present)
- `error` — Error message (if handler returned error)

### Custom Fields

Add custom fields in handlers:

```go
router.GET("/api/users/:id", func(ctx *chain.Context) error {
    userID := ctx.GetParam("id")
    
    slog.Info(
        "fetching user",
        "user_id", userID,
        "request_id", logger.GetRequestID(ctx),
    )
    
    return nil
})
```

## Troubleshooting

### No Log Output

**Cause:** Logger not registered or using wrong log level.

**Solution:**
```go
// Ensure logger is registered
router.Use(logger.New())

// Check log level
router.Use(logger.New(logger.Config{
    Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })),
}))
```

### Request ID Not Available

**Cause:** `GenerateRequestID` is false or header name mismatch.

**Solution:**
```go
router.Use(logger.New(logger.Config{
    GenerateRequestID: true,
    RequestIDHeader:   "X-Request-ID",
}))
```

### Slow Request Warnings Not Showing

**Cause:** `LatencyThreshold` not set or threshold too high.

**Solution:**
```go
router.Use(logger.New(logger.Config{
    LatencyThreshold: 500 * time.Millisecond,
}))
```

## See Also

- [log/slog Documentation](https://pkg.go.dev/log/slog)
- [Chain Middleware Overview](../README.md)
- [Chain Router Documentation](../../README.md)
