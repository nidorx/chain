# Chain Middleware

Middleware for the Chain HTTP router. Each middleware package provides specific functionality that can be easily integrated into your Chain applications.

## Available Middleware

### 1. CORS Middleware (`cors/`)

Cross-Origin Resource Sharing (CORS) middleware with comprehensive configuration options.

**Features:**
- Configurable allowed origins
- Wildcard origin support (e.g., `http://*.example.com`)
- Regex-based origin matching
- Custom origin validation functions
- Preflight request handling
- Credentials support
- Private network access support
- Custom headers and methods configuration

**Installation:**
```go
import "github.com/nidorx/chain/middlewares/cors"
```

**Basic Usage:**
```go
router := chain.New()

// Allow all origins
router.Use(cors.Default())

// Custom configuration
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:          12 * time.Hour,
}))
```

**Advanced Usage:**
```go
// Wildcard origins
router.Use(cors.New(cors.Config{
    AllowOrigins:  []string{"http://*.example.com"},
    AllowWildcard: true,
}))

// Custom validation function
router.Use(cors.New(cors.Config{
    AllowOriginFunc: func(origin string) bool {
        return origin == "http://allowed.com"
    },
}))

// With context
router.Use(cors.New(cors.Config{
    AllowOriginWithContextFunc: func(ctx *chain.Context, origin string) bool {
        // Access request context to make decisions
        return strings.HasSuffix(origin, ".example.com")
    },
}))
```

---

### 2. Logger Middleware (`logger/`)

Structured logging middleware using Go's `log/slog` package.

**Features:**
- Request/response logging
- Duration tracking
- Status code logging
- Request ID generation
- Custom log formats
- Skip paths and prefixes
- Configurable log levels based on status codes
- Latency threshold warnings

**Installation:**
```go
import "github.com/nidorx/chain/middlewares/logger"
```

**Basic Usage:**
```go
router := chain.New()

// Default logging
router.Use(logger.New())

// Custom configuration
router.Use(logger.New(logger.Config{
    Format: logger.FormatDefault,
    Logger: slog.Default(),
}))
```

**Advanced Usage:**
```go
// Skip health checks
router.Use(logger.New(logger.Config{
    SkipPaths: []string{"/health", "/ping"},
}))

// Skip path prefixes
router.Use(logger.New(logger.Config{
    SkipPathPrefixes: []string{"/static/", "/assets/"},
}))

// Custom format
router.Use(logger.New(logger.Config{
    Format:       logger.FormatCustom,
    CustomFormat: "%{method} %{path} %{status} %{latency}",
}))

// JSON format
router.Use(logger.New(logger.Config{
    Format: logger.FormatJSON,
}))

// Custom request ID header
router.Use(logger.New(logger.Config{
    RequestIDHeader: "X-Request-ID",
}))

// Latency warnings
router.Use(logger.New(logger.Config{
    LatencyThreshold: 500 * time.Millisecond,
}))

// Custom log levels
router.Use(logger.New(logger.Config{
    StatusLevelFunc: func(status int) slog.Level {
        if status >= 500 {
            return slog.LevelError
        }
        if status >= 400 {
            return slog.LevelWarn
        }
        return slog.LevelInfo
    },
}))
```

**Request ID:**
```go
// Get request ID in handler
router.GET("/test", func(ctx *chain.Context) error {
    requestID := logger.GetRequestID(ctx)
    // Use requestID for tracing
    return nil
})
```

---

### 3. Recovery Middleware (`recovery/`)

Panic recovery middleware with stack trace support.

**Features:**
- Automatic panic recovery
- Stack trace logging
- Custom recovery handlers
- Configurable stack trace size
- Optional panic logging disable

**Installation:**
```go
import "github.com/nidorx/chain/middlewares/recovery"
```

**Basic Usage:**
```go
router := chain.New()

// Default recovery (recommended as first middleware)
router.Use(recovery.New())
```

**Advanced Usage:**
```go
// Custom recovery handler
router.Use(recovery.New(recovery.Config{
    RecoveryHandler: func(ctx *chain.Context, err any) {
        ctx.Json(map[string]string{
            "error": "internal server error",
        })
    },
}))

// Disable stack trace printing
router.Use(recovery.New(recovery.Config{
    PrintStack: false,
}))

// Custom stack size
router.Use(recovery.New(recovery.Config{
    StackSize: 8192,
}))
```

**Best Practice:**
Always use recovery as the **first** middleware in your chain to catch panics from all subsequent middleware and handlers:

```go
router := chain.New()

// Recovery should be first
router.Use(recovery.New())

// Then other middleware
router.Use(logger.New())
router.Use(cors.Default())
```

---

### 4. Limiter Middleware (`limiter/`)

Request body size limiting middleware.

**Features:**
- Configurable max body size
- Automatic 413 response on limit exceeded
- Custom error handlers
- Method-specific skipping
- Integration with `ctx.BodyBytes()`

**Installation:**
```go
import "github.com/nidorx/chain/middlewares/limiter"
```

**Basic Usage:**
```go
router := chain.New()

// Default 10MB limit
router.Use(limiter.New())

// Custom limit
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 1 << 20, // 1MB
}))
```

**Advanced Usage:**
```go
// Very small limit
router.Use(limiter.MaxBytes(1024)) // 1KB

// Custom error response
router.Use(limiter.New(limiter.Config{
    MaxBodySize: 5 << 20, // 5MB
    ErrorHandler: func(ctx *chain.Context) {
        ctx.Json(map[string]string{
            "error": "request body too large",
        }, 413)
    },
}))

// Skip certain methods
router.Use(limiter.New(limiter.Config{
    MaxBodySize:   10 << 20,
    SkipMethods:   []string{"GET", "HEAD", "OPTIONS"},
}))
```

**Handler Integration:**
The middleware works automatically with `ctx.BodyBytes()`:

```go
router.POST("/upload", func(ctx *chain.Context) error {
    body, err := ctx.BodyBytes()
    if err != nil {
        return err // Will be caught by limiter middleware
    }
    // Process body
    return nil
})
```

---

### 5. Session Middleware (`session/`)

Cookie-based session management with encryption support.

**Features:**
- Encrypted cookie storage
- HMAC signing
- Key rotation support
- Multiple session managers
- Session lifecycle management

**Installation:**
```go
import "github.com/nidorx/chain/middlewares/session"
```

**Basic Usage:**
```go
// Set secret key
chain.SetSecretKeyBase("your-secret-key-base")

// Configure middleware
sessionManager := &session.Manager{
    Store: &session.Cookie{},
    Config: session.Config{
        Key:      "_session",
        MaxAge:   86400,
        HttpOnly: true,
        Secure:   true,
    },
}

router.Use("/*", sessionManager)

// Use in handlers
router.GET("/login", func(ctx *chain.Context) error {
    sess, err := session.Fetch(ctx)
    if err != nil {
        return err
    }
    sess.Put("user_id", "123")
    ctx.OK("logged in")
    return nil
})
```

**See:** [Session Middleware Documentation](session/README.md)

---

## Middleware Registration

### Global Middleware
```go
router.Use(middleware)
```

### Path-Scoped Middleware
```go
router.Use("/api/*", middleware)
```

### Method + Path Scoped
```go
router.Use("GET", "/api/*", middleware)
```

### Group Scoped
```go
api := router.Group("/api")
api.Use(middleware)
```

---

## Middleware Order

The order of middleware registration matters. Middlewares are executed in the order they are registered:

```go
router := chain.New()

// 1. Recovery (catches panics from everything below)
router.Use(recovery.New())

// 2. Logger (logs all requests)
router.Use(logger.New())

// 3. CORS (handles cross-origin requests)
router.Use(cors.Default())

// 4. Limiter (limits body size)
router.Use(limiter.New())

// 5. Session (manages sessions)
router.Use(sessionManager)

// Routes
router.GET("/test", handler)
```

---

## Writing Custom Middleware

Middleware signature:
```go
type MiddlewareFunc func(ctx *chain.Context, next func() error) error
```

Example:
```go
func MyMiddleware() chain.MiddlewareFunc {
    return func(ctx *chain.Context, next func() error) error {
        // Before handler
        start := time.Now()
        
        // Call next middleware/handler
        err := next()
        
        // After handler
        log.Printf("Request took %v", time.Since(start))
        
        return err
    }
}

router.Use(MyMiddleware())
```

---

## Testing

All middleware packages include comprehensive test suites:

```bash
# Run all middleware tests
go test ./middlewares/...

# Run specific middleware tests
go test ./middlewares/cors/...
go test ./middlewares/logger/...
go test ./middlewares/recovery/...
go test ./middlewares/limiter/...
go test ./middlewares/session/...
```

---

## References

- [gin-contrib/cors](https://github.com/gin-contrib/cors) - Inspiration for CORS middleware
- [Chain Router Documentation](../../README.md)
- [Chain API Reference](../../docs/03-api-reference.md)
