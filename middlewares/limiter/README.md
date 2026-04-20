# Limiter Middleware

Request body size limiting middleware for Chain.

## Overview

The limiter middleware restricts the size of request bodies to prevent resource exhaustion attacks, such as large body denial-of-service (DoS) attacks. It integrates with Chain's context system and works seamlessly with `ctx.BodyBytes()` to enforce size limits.

## Features

- **Configurable limits** — Set custom max body size per route or globally
- **Automatic enforcement** — Works with `ctx.BodyBytes()` out of the box
- **Custom error handlers** — Provide custom error responses
- **Method skipping** — Skip limits for certain HTTP methods
- **Content-Length checking** — Early rejection based on Content-Length header
- **Multiple status codes** — Configurable error status codes

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/nidorx/chain"
    "github.com/nidorx/chain/middlewares/limiter"
    "net/http"
)

func main() {
    router := chain.New()
    
    // Default 10MB limit
    router.Use(limiter.New())
    
    router.POST("/api/data", func(ctx *chain.Context) error {
        body, err := ctx.BodyBytes()
        if err != nil {
            return err // Will be caught by limiter middleware
        }
        // Process body
        ctx.OK("received")
        return nil
    })
    
    http.ListenAndServe(":8080", router)
}
```

## Configuration

### Config Struct

```go
type Config struct {
    // Maximum allowed request body size in bytes (default: 10MB)
    MaxBodySize int64
    
    // Custom error handler function
    ErrorHandler func(ctx *chain.Context)
    
    // HTTP methods to skip (default: GET, HEAD, OPTIONS)
    SkipMethods []string
    
    // HTTP status code for errors (default: 413)
    StatusCode int
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `MaxBodySize` | `int64` | `10485760` (10MB) | Max body size in bytes |
| `ErrorHandler` | `func(*chain.Context)` | `nil` | Custom error handler |
| `SkipMethods` | `[]string` | `["GET", "HEAD", "OPTIONS"]` | Methods to skip |
| `StatusCode` | `int` | `413` | Error status code |

## Usage Examples

### Default Limit (10MB)

```go
router.Use(limiter.New())
```

### Custom Limit

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20, // 1MB
}))
```

### Very Small Limit

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1024, // 1KB
}))
```

### Convenience Function

```go
// 1MB limit
router.Use(limiter.MaxBytes(1 << 20))

// 5MB limit
router.Use(limiter.MaxBytes(5 << 20))

// 100KB limit
router.Use(limiter.MaxBytes(100 << 10))
```

### Custom Error Handler

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20, // 1MB
    ErrorHandler: func(ctx *chain.Context) {
        ctx.Json(map[string]any{
            "success": false,
            "error": map[string]string{
                "code":    "PAYLOAD_TOO_LARGE",
                "message": "Request body too large",
            },
        })
    },
}))
```

### Custom Status Code

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20,
    StatusCode:  400, // Use 400 instead of 413
}))
```

### Skip Certain Methods

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 10 << 20, // 10MB
    SkipMethods: []string{
        http.MethodGet,
        http.MethodHead,
        http.MethodOptions,
    },
}))
```

### Per-Route Limits

```go
// Global limit: 10MB
router.Use(limiter.New())

// Specific route with smaller limit
uploadGroup := router.Group("/upload")
uploadGroup.Use(limiter.New(limiter.Config{
    MaxBodySize: 50 << 20, // 50MB for uploads
}))

uploadGroup.POST("/file", fileUploadHandler)

// API with smaller limit
apiGroup := router.Group("/api")
apiGroup.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20, // 1MB for API
}))

apiGroup.POST("/data", dataHandler)
```

## Integration Examples

### File Upload

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 50 << 20, // 50MB
    ErrorHandler: func(ctx *chain.Context) {
        ctx.Status(413, map[string]string{
            "error": "File too large. Maximum size is 50MB.",
        })
    },
}))

router.POST("/upload", func(ctx *chain.Context) error {
    file, header, err := ctx.Request.FormFile("file")
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Check file size
    if header.Size > 50<<20 {
        return fmt.Errorf("file too large")
    }
    
    // Save file
    // ...
    
    ctx.OK("uploaded")
    return nil
})
```

### JSON API

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20, // 1MB for JSON API
}))

router.POST("/api/users", func(ctx *chain.Context) error {
    var user User
    if err := ctx.BindJSON(&user); err != nil {
        return err
    }
    
    // Save user
    // ...
    
    ctx.Created(user)
    return nil
})
```

### GraphQL

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 100 << 10, // 100KB for GraphQL
}))

router.POST("/graphql", func(ctx *chain.Context) error {
    var query GraphQLQuery
    if err := ctx.BindJSON(&query); err != nil {
        return err
    }
    
    // Execute query
    // ...
    
    ctx.Json(result)
    return nil
})
```

### With Logger and Recovery

```go
router := chain.New()

// Standard middleware order
router.Use(recovery.New())
router.Use(logger.New())
router.Use(limiter.New())

router.POST("/api/data", func(ctx *chain.Context) error {
    body, err := ctx.BodyBytes()
    if err != nil {
        return err
    }
    
    slog.Info("received request", "size", len(body))
    
    return nil
})
```

## How It Works

### Enforcement Flow

```
Request arrives
    │
    ▼
Limiter middleware
    │
    ├─ Check if method should be skipped (GET, HEAD, OPTIONS)
    │   └─ If yes, skip to next middleware
    │
    ├─ Wrap request body with http.MaxBytesReader
    │
    ├─ Set max body size in context
    │
    ▼
Next middleware/handler
    │
    ├─ Reads body (via ctx.BodyBytes() or directly)
    │   └─ If exceeds limit, error is returned
    │
    ▼
Limiter middleware (after next returns)
    │
    ├─ Check if error is body size error
    │   ├─ If yes, return custom error or 413
    │   └─ If no, pass error through
    │
    ▼
Response sent
```

### Body Reading

The middleware wraps `ctx.Request.Body` with `http.MaxBytesReader`. When the body is read (via `ctx.BodyBytes()` or directly), it enforces the limit:

```go
// This will enforce the limit
body, err := ctx.BodyBytes()

// This will also enforce (direct access)
body, err := io.ReadAll(ctx.Request.Body)
```

### Content-Length Check

If the `Content-Length` header is present and exceeds the limit, the request is rejected immediately without reading the body:

```go
// Request with Content-Length: 20000000 (20MB)
// Middleware with MaxBodySize: 10485760 (10MB)
// → Immediate 413 rejection
```

## API Reference

### Functions

```go
// DefaultConfig returns a default configuration
func DefaultConfig() Config

// New creates limiter middleware
func New(config ...Config) chain.MiddlewareFunc

// MaxBytes creates limiter with specific byte limit
func MaxBytes(maxSize int64) chain.MiddlewareFunc
```

### Types

```go
type Config struct {
    MaxBodySize    int64
    ErrorHandler   func(ctx *chain.Context)
    SkipMethods    []string
    StatusCode     int
}
```

### Constants

```go
const DefaultMaxBodySize = 10 << 20 // 10MB
```

## Common Size Limits

| Use Case | Recommended Limit | Example |
|----------|-------------------|---------|
| JSON API | 1MB | `1 << 20` |
| Form data | 5MB | `5 << 20` |
| File upload | 50MB | `50 << 20` |
| GraphQL | 100KB | `100 << 10` |
| Webhook | 64KB | `64 << 10` |
| Text data | 10KB | `10 << 10` |

## Error Handling

### Default Error Response

```json
{
  "error": "request body too large",
  "max_size": 10485760
}
```

Status: `413 Payload Too Large`

### Custom Error Response

```go
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20,
    ErrorHandler: func(ctx *chain.Context) {
        ctx.Status(413, map[string]any{
            "success": false,
            "error": {
                "code": "PAYLOAD_TOO_LARGE",
                "message": "Request body exceeds the maximum allowed size of 1MB",
                "max_size": 1 << 20,
            },
        })
    },
}))
```

### Handler-Level Error Handling

```go
router.POST("/api/data", func(ctx *chain.Context) error {
    body, err := ctx.BodyBytes()
    if err != nil {
        // Handle error in handler
        if strings.Contains(err.Error(), "request body too large") {
            ctx.Status(413, map[string]string{
                "error": "Data too large",
            })
            return nil
        }
        return err
    }
    
    // Process body
    return nil
})
```

## Performance Considerations

### Overhead

The limiter middleware adds minimal overhead:
- MaxBytesReader wrapping: ~1μs
- Content-Length check: ~100ns
- Context storage: ~50ns

### Early Rejection

The middleware checks `Content-Length` first for early rejection:

```go
// Request with Content-Length: 50MB
// Limit: 1MB
// → Rejected immediately without reading body
```

This saves bandwidth and processing time.

### Skip Unnecessary Methods

Skip methods that typically don't have bodies:

```go
router.Use(limiter.New(limiter.Config{
    SkipMethods: []string{
        http.MethodGet,
        http.MethodHead,
        http.MethodOptions,
    },
}))
```

## Testing

### Test Limit Enforcement

```go
func TestLimiter(t *testing.T) {
    router := chain.New()
    router.Use(limiter.New(limiter.Config{
        MaxBodySize: 100,
    }))
    
    router.POST("/test", func(ctx *chain.Context) error {
        _, err := ctx.BodyBytes()
        return err
    })
    
    // Test with large body
    largeBody := make([]byte, 200)
    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/test", bytes.NewReader(largeBody))
    router.ServeHTTP(w, req)
    
    if w.Code != 413 {
        t.Errorf("Expected 413, got %d", w.Code)
    }
}
```

### Test Small Body

```go
func TestLimiterSmallBody(t *testing.T) {
    router := chain.New()
    router.Use(limiter.New(limiter.Config{
        MaxBodySize: 100,
    }))
    
    router.POST("/test", func(ctx *chain.Context) error {
        ctx.OK("ok")
        return nil
    })
    
    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("test")))
    router.ServeHTTP(w, req)
    
    if w.Code != 200 {
        t.Errorf("Expected 200, got %d", w.Code)
    }
}
```

## Troubleshooting

### Body Not Limited

**Cause:** Body not being read in handler.

**Solution:**
```go
// ✅ GOOD - BodyBytes enforces limit
router.POST("/upload", func(ctx *chain.Context) error {
    body, err := ctx.BodyBytes()
    if err != nil {
        return err
    }
    // Process body
    return nil
})

// ❌ BAD - Body not read, limit not enforced
router.POST("/upload", func(ctx *chain.Context) error {
    // Body not read
    ctx.OK("ok")
    return nil
})
```

### Limit Not Applied to Certain Routes

**Cause:** Global limiter doesn't cover all routes.

**Solution:**
```go
// Apply to specific group
apiGroup := router.Group("/api")
apiGroup.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20,
}))
```

### GET Requests Being Limited

**Cause:** SkipMethods not configured.

**Solution:**
```go
router.Use(limiter.New(limiter.Config{
    SkipMethods: []string{
        http.MethodGet,
        http.MethodHead,
        http.MethodOptions,
    },
}))
```

## Security Considerations

### Always Set Reasonable Limits

```go
// ❌ BAD - No limit
router.POST("/upload", handler)

// ✅ GOOD - Reasonable limit
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 50 << 20, // 50MB
}))
```

### Different Limits for Different Endpoints

```go
// Public API - small limit
public.Use(limiter.New(limiter.Config{
    MaxBodySize: 100 << 10, // 100KB
}))

// Admin API - larger limit
admin.Use(limiter.New(limiter.Config{
    MaxBodySize: 10 << 20, // 10MB
}))
```

### Combine with Other Security Measures

```go
router.Use(recovery.New())
router.Use(logger.New())
router.Use(limiter.New()) // Body size limit
router.Use(auth.New())    // Authentication

// Multiple layers of protection
```

## See Also

- [http.MaxBytesReader Documentation](https://pkg.go.dev/net/http#MaxBytesReader)
- [Chain Middleware Overview](../README.md)
- [Chain Context API](../../docs/03-api-reference.md)
