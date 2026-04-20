# CORS Middleware

Cross-Origin Resource Sharing (CORS) middleware for Chain with comprehensive configuration options.

## Overview

The CORS middleware handles Cross-Origin Resource Sharing, allowing you to control which origins can access your API, what methods and headers are allowed, and how preflight requests are handled. It integrates seamlessly with Chain's middleware system.

## Features

- **Configurable origins** — Specify allowed origins explicitly or use wildcards
- **Wildcard support** — Pattern matching for origins (e.g., `http://*.example.com`)
- **Regex matching** — Regular expression-based origin validation
- **Custom validation** — Provide custom functions to validate origins
- **Preflight handling** — Automatic handling of OPTIONS requests
- **Credentials support** — Allow cookies and HTTP authentication
- **Private network access** — Support for private network requests
- **Flexible headers** — Configure allowed and exposed headers
- **Max age control** — Control preflight result caching duration

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/nidorx/chain"
    "github.com/nidorx/chain/middlewares/cors"
    "net/http"
)

func main() {
    router := chain.New()
    
    // Allow all origins (development only)
    router.Use(cors.Default())
    
    router.GET("/api/data", func(ctx *chain.Context) error {
        ctx.Json(map[string]string{"message": "Hello World"})
        return nil
    })
    
    http.ListenAndServe(":8080", router)
}
```

### Production Configuration

```go
router := chain.New()

// Configure CORS for production
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{
        "https://example.com",
        "https://app.example.com",
    },
    AllowMethods: []string{
        "GET",
        "POST", 
        "PUT",
        "DELETE",
        "OPTIONS",
    },
    AllowHeaders: []string{
        "Origin",
        "Content-Type",
        "Accept",
        "Authorization",
    },
    AllowCredentials: true,
    MaxAge:          12 * time.Hour,
}))
```

## Configuration

### Config Struct

```go
type Config struct {
    // Allow all origins (conflicts with AllowOrigins)
    AllowAllOrigins bool
    
    // List of allowed origins
    AllowOrigins []string
    
    // Custom origin validation function
    AllowOriginFunc func(origin string) bool
    
    // Custom origin validation with request context
    AllowOriginWithContextFunc func(c *chain.Context, origin string) bool
    
    // Allowed HTTP methods (default: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
    AllowMethods []string
    
    // Allow private network access
    AllowPrivateNetwork bool
    
    // Allowed HTTP headers for CORS requests
    AllowHeaders []string
    
    // Allow credentials (cookies, HTTP auth)
    AllowCredentials bool
    
    // Headers safe to expose to CORS API specification
    ExposeHeaders []string
    
    // Preflight cache duration (default: 12 hours)
    MaxAge time.Duration
    
    // Enable wildcard origin parsing
    AllowWildcard bool
    
    // Allow browser extension schemas
    AllowBrowserExtensions bool
    
    // Custom schemas (e.g., tauri://)
    CustomSchemas []string
    
    // Allow WebSocket schemas (ws://, wss://)
    AllowWebSockets bool
    
    // Allow file:// schema (dangerous)
    AllowFiles bool
    
    // Custom OPTIONS response status code
    OptionsResponseStatusCode int
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AllowAllOrigins` | `bool` | `false` | Allow all origins (sets `*`) |
| `AllowOrigins` | `[]string` | `[]` | List of allowed origins |
| `AllowOriginFunc` | `func(string) bool` | `nil` | Custom origin validation function |
| `AllowOriginWithContextFunc` | `func(*chain.Context, string) bool` | `nil` | Custom validation with request context |
| `AllowMethods` | `[]string` | 7 methods | Allowed HTTP methods |
| `AllowHeaders` | `[]string` | 3 headers | Allowed request headers |
| `AllowCredentials` | `bool` | `false` | Allow credentials |
| `ExposeHeaders` | `[]string` | `[]` | Headers to expose to client |
| `MaxAge` | `time.Duration` | `12h` | Preflight cache duration |
| `AllowWildcard` | `bool` | `false` | Enable wildcard patterns |
| `AllowPrivateNetwork` | `bool` | `false` | Allow private network access |
| `OptionsResponseStatusCode` | `int` | `204` | OPTIONS response status code |

## Usage Examples

### Allow All Origins

```go
// Method 1: Using Default()
router.Use(cors.Default())

// Method 2: Using Config
router.Use(cors.New(cors.Config{
    AllowAllOrigins: true,
}))
```

**Warning:** Only use this in development. In production, always specify allowed origins.

### Specific Origins

```go
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{
        "https://example.com",
        "https://admin.example.com",
    },
}))
```

### Wildcard Origins

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:  []string{"http://*.example.com"},
    AllowWildcard: true,
}))
```

This will match:
- `http://app.example.com` ✅
- `http://api.example.com` ✅
- `http://example.com` ✅
- `https://app.example.com` ❌ (different protocol)

### Multiple Wildcards

```go
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{
        "http://*.example.com",
        "https://*.example.com",
    },
    AllowWildcard: true,
}))
```

### Regex Origin Matching

```go
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{
        "/^https://.*\\.example\\.com$/",
    },
}))
```

This uses regex to match origins with any subdomain of `example.com` over HTTPS.

### Custom Origin Validation

```go
router.Use(cors.New(cors.Config{
    AllowOriginFunc: func(origin string) bool {
        // Custom logic
        return strings.HasSuffix(origin, ".mycompany.com")
    },
}))
```

### Custom Validation with Context

```go
router.Use(cors.New(cors.Config{
    AllowOriginWithContextFunc: func(ctx *chain.Context, origin string) bool {
        // Access request context to make decisions
        path := ctx.URL().Path
        
        // Different rules for different paths
        if strings.HasPrefix(path, "/public") {
            return true // Allow all for public paths
        }
        
        // Check against database or cache
        return isOriginAllowed(origin)
    },
}))
```

### Credentials Support

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://example.com"},
    AllowCredentials: true,
}))
```

When `AllowCredentials` is `true`, `AllowAllOrigins` cannot be used. You must specify explicit origins.

### Custom Headers

```go
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{"https://example.com"},
    AllowHeaders: []string{
        "Origin",
        "Content-Type",
        "Accept",
        "Authorization",
        "X-Requested-With",
        "X-Custom-Header",
    },
    ExposeHeaders: []string{
        "X-Request-Id",
        "X-Rate-Limit",
    },
}))
```

### Preflight Cache

```go
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{"https://example.com"},
    MaxAge:       24 * time.Hour, // Cache preflight for 24 hours
}))
```

### Private Network Access

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:        []string{"https://example.com"},
    AllowPrivateNetwork: true,
}))
```

This adds the `Access-Control-Allow-Private-Network: true` header for Chrome's Private Network Access requests.

### Browser Extensions

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:           []string{"chrome-extension://abcdefghijk"},
    AllowBrowserExtensions: true,
}))
```

Supports:
- `chrome-extension://`
- `safari-extension://`
- `moz-extension://`
- `ms-browser-extension://`

### WebSocket Support

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:    []string{"ws://example.com", "wss://example.com"},
    AllowWebSockets: true,
}))
```

### Custom Schemas

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:    []string{"tauri://localhost"},
    CustomSchemas:   []string{"tauri://"},
}))
```

### Custom OPTIONS Status Code

For older browsers/clients:

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:            []string{"https://example.com"},
    OptionsResponseStatusCode: http.StatusOK,
}))
```

## Helper Methods

### Add Allow Methods

```go
config := cors.DefaultConfig()
config.AddAllowMethods("CONNECT", "TRACE")
router.Use(cors.New(config))
```

### Add Allow Headers

```go
config := cors.DefaultConfig()
config.AddAllowHeaders("Authorization", "X-API-Key")
router.Use(cors.New(config))
```

### Add Expose Headers

```go
config := cors.DefaultConfig()
config.AddExposeHeaders("X-Request-Id", "X-Rate-Limit")
router.Use(cors.New(config))
```

## API Reference

### Functions

```go
// DefaultConfig returns a generic default configuration
func DefaultConfig() Config

// Default returns CORS middleware with all origins allowed
func Default() chain.Handle

// New returns CORS middleware with custom configuration
func New(config Config) chain.Handle
```

### Types

```go
type Config struct {
    AllowAllOrigins            bool
    AllowOrigins               []string
    AllowOriginFunc            func(origin string) bool
    AllowOriginWithContextFunc func(c *chain.Context, origin string) bool
    AllowMethods               []string
    AllowPrivateNetwork        bool
    AllowHeaders               []string
    AllowCredentials           bool
    ExposeHeaders              []string
    MaxAge                     time.Duration
    AllowWildcard              bool
    AllowBrowserExtensions     bool
    CustomSchemas              []string
    AllowWebSockets            bool
    AllowFiles                 bool
    OptionsResponseStatusCode  int
}
```

## CORS Headers Explained

### Response Headers Set by Middleware

| Header | When Set | Description |
|--------|----------|-------------|
| `Access-Control-Allow-Origin` | Always | Allowed origin or `*` |
| `Access-Control-Allow-Methods` | Preflight | Allowed HTTP methods |
| `Access-Control-Allow-Headers` | Preflight | Allowed request headers |
| `Access-Control-Allow-Credentials` | If enabled | Allow credentials |
| `Access-Control-Max-Age` | Preflight | Cache duration (seconds) |
| `Access-Control-Expose-Headers` | Always | Headers safe to expose |
| `Access-Control-Allow-Private-Network` | Preflight | Allow private network |
| `Vary` | Always | Cache variation headers |

### Request Headers Checked

| Header | Purpose |
|--------|---------|
| `Origin` | Determines if request is CORS |
| `Access-Control-Request-Method` | Method requested for preflight |
| `Access-Control-Request-Headers` | Headers requested for preflight |

## Integration Examples

### With Authentication

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://app.example.com"},
    AllowCredentials: true,
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
}))

router.Use(authMiddleware)

router.GET("/api/protected", func(ctx *chain.Context) error {
    // Authenticated and CORS-enabled
    return nil
})
```

### With Logging

```go
// Order matters: recovery → logger → CORS → routes
router.Use(recovery.New())
router.Use(logger.New())
router.Use(cors.Default())

router.GET("/api/data", func(ctx *chain.Context) error {
    ctx.Json(data)
    return nil
})
```

### Multiple Route Groups

```go
// Public API - allow all
public := router.Group("/public")
public.Use(cors.Default())
public.GET("/data", publicHandler)

// Private API - restricted
private := router.Group("/private")
private.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://admin.example.com"},
    AllowCredentials: true,
}))
private.GET("/users", adminHandler)
```

## Security Considerations

### Never Use AllowAllOrigins in Production

```go
// ❌ BAD - Production
router.Use(cors.Default())

// ✅ GOOD - Production
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{"https://yourdomain.com"},
}))
```

### Be Careful with Credentials

When `AllowCredentials` is `true`:
- Cookies are sent with requests
- `AllowAllOrigins` cannot be used
- Must specify exact origins

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://trusted.com"},
    AllowCredentials: true,
}))
```

### Validate Custom Origins

When using `AllowOriginFunc`, ensure proper validation:

```go
router.Use(cors.New(cors.Config{
    AllowOriginFunc: func(origin string) bool {
        // ❌ BAD - allows any origin
        return true
        
        // ✅ GOOD - validates against list
        allowedDomains := []string{"example.com", "trusted.com"}
        for _, domain := range allowedDomains {
            if strings.HasSuffix(origin, domain) {
                return true
            }
        }
        return false
    },
}))
```

### Avoid file:// Schema

```go
// ❌ DANGEROUS
router.Use(cors.New(cors.Config{
    AllowOrigins: []string{"file://"},
    AllowFiles:   true,
}))
```

Only use `AllowFiles` when absolutely necessary and you're certain of the security implications.

## Troubleshooting

### "No 'Access-Control-Allow-Origin' header" Error

**Cause:** Origin not in allowed list or middleware not registered.

**Solution:**
```go
// Check origin is allowed
config := cors.Config{
    AllowOrigins: []string{"http://your-origin.com"},
}
router.Use(cors.New(config))
```

### Preflight Requests Failing

**Cause:** Missing methods or headers in configuration.

**Solution:**
```go
router.Use(cors.New(cors.Config{
    AllowOrigins:   []string{"https://example.com"},
    AllowMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:   []string{"Authorization", "Content-Type"},
    MaxAge:         12 * time.Hour,
}))
```

### Credentials Not Being Sent

**Cause:** `AllowCredentials` not enabled or `AllowAllOrigins` is true.

**Solution:**
```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://example.com"}, // Cannot be *
    AllowCredentials: true,
}))
```

### Wildcard Not Working

**Cause:** `AllowWildcard` not enabled.

**Solution:**
```go
router.Use(cors.New(cors.Config{
    AllowOrigins:  []string{"http://*.example.com"},
    AllowWildcard: true,
}))
```

## See Also

- [MDN CORS Documentation](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [Chain Router Documentation](../../README.md)
- [Chain Middleware Overview](../README.md)
