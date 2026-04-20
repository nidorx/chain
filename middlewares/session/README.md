# Session Middleware

Cookie-based session management for Chain with optional encryption and signing.

## Overview

The session middleware provides a secure, flexible session management system that stores session data in encrypted cookies. It integrates seamlessly with Chain's crypto utilities for key derivation and encryption.

## Features

- **Cookie-based storage** — Sessions are stored as encrypted cookies on the client
- **Encryption support** — Uses AES-GCM encryption via `chain.Crypto()`
- **Message signing** — HMAC-based signing to prevent tampering
- **Key rotation** — Integrates with Chain's Keyring for seamless key rotation
- **Session lifecycle** — Automatic session creation, renewal, and destruction
- **Thread-safe** — Safe for concurrent access

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/nidorx/chain"
    "github.com/nidorx/chain/middlewares/session"
    "net/http"
)

func main() {
    router := chain.New()
    
    // Set secret key for encryption
    chain.SetSecretKeyBase("your-secret-key-base")
    
    // Configure session middleware
    sessionManager := &session.Manager{
        Store: &session.Cookie{},
        Config: session.Config{
            Key:      "_chain_session",
            MaxAge:   86400, // 24 hours
            HttpOnly: true,
            Secure:   true,
            SameSite: http.SameSiteStrictMode,
        },
    }
    
    // Register middleware globally
    router.Use("/*", sessionManager)
    
    router.GET("/login", func(ctx *chain.Context) error {
        sess, err := session.Fetch(ctx)
        if err != nil {
            return err
        }
        sess.Put("user_id", "123")
        sess.Put("role", "admin")
        ctx.OK("logged in")
        return nil
    })
    
    router.GET("/profile", func(ctx *chain.Context) error {
        sess, err := session.Fetch(ctx)
        if err != nil {
            return err
        }
        userID := sess.Get("user_id")
        ctx.Json(map[string]any{"user_id": userID})
        return nil
    })
    
    router.GET("/logout", func(ctx *chain.Context) error {
        sess, err := session.Fetch(ctx)
        if err != nil {
            return err
        }
        sess.Destroy()
        ctx.OK("logged out")
        return nil
    })
    
    http.ListenAndServe(":8080", router)
}
```

### Multiple Session Managers

You can have multiple session managers with different keys for different parts of your application:

```go
// Admin session
adminSession := &session.Manager{
    Store: &session.Cookie{},
    Config: session.Config{
        Key:    "_admin_session",
        MaxAge: 3600, // 1 hour
    },
}

// User session
userSession := &session.Manager{
    Store: &session.Cookie{},
    Config: session.Config{
        Key:    "_user_session",
        MaxAge: 86400, // 24 hours
    },
}

router.Use("/admin/*", adminSession)
router.Use("/user/*", userSession)

// Fetch specific session by key
router.GET("/admin/dashboard", func(ctx *chain.Context) error {
    sess, err := session.FetchByKey(ctx, "_admin_session")
    if err != nil {
        return err
    }
    // Use admin session
    return nil
})
```

## Configuration

### Config Struct

```go
type Config struct {
    Key        string        // Session cookie key (required)
    Path       string        // Cookie path (default: "/")
    Domain     string        // Cookie domain
    MaxAge     int           // Max age in seconds (default: 86400)
    Secure     bool          // Secure flag (HTTPS only)
    HttpOnly   bool          // HTTP only flag (no JavaScript access)
    SameSite   http.SameSite // SameSite policy (Lax, Strict, None)
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Key` | `string` | (required) | Name of the session cookie |
| `Path` | `string` | `"/"` | Cookie path scope |
| `Domain` | `string` | `""` | Cookie domain scope |
| `MaxAge` | `int` | `86400` | Session lifetime in seconds |
| `Secure` | `bool` | `true` | Only send over HTTPS |
| `HttpOnly` | `bool` | `true` | Prevent JavaScript access |
| `SameSite` | `http.SameSite` | `StrictMode` | CSRF protection level |

## Session API

### Fetching Sessions

```go
// Fetch global session (if configured)
sess, err := session.Fetch(ctx)

// Fetch specific session by key
sess, err := session.FetchByKey(ctx, "_my_session")
```

### Session Methods

```go
// Get a value
value := sess.Get("key")

// Check if key exists
exists := sess.Exist("key")

// Set a value
sess.Put("key", "value")

// Delete a key
sess.Delete("key")

// Get all data as map
data := sess.GetMap()

// Clear all data (session still exists)
sess.Clear()

// Renew session ID (prevents session fixation)
sess.Renew()

// Destroy session completely (cookie removed)
sess.Destroy()

// Ignore all changes made in this request
sess.IgnoreChanges()
```

### Session States

The session tracks its state internally:

| State | Description |
|-------|-------------|
| `none` | No changes made |
| `write` | Data modified, will be saved |
| `drop` | Session should be deleted |
| `renew` | Session ID should be regenerated |
| `ignore` | Ignore all changes in this request |

## Store Interface

You can implement custom storage backends by implementing the `Store` interface:

```go
type Store interface {
    Name() string
    Init(config Config, router *chain.Router) error
    Get(ctx *chain.Context, rawCookie string) (sid string, data map[string]any)
    Put(ctx *chain.Context, sid string, data map[string]any) (rawCookie string, err error)
    Delete(ctx *chain.Context, sid string)
}
```

### Built-in Stores

#### Cookie Store

The default store that encrypts session data in cookies:

```go
&session.Cookie{}
```

Features:
- Automatic encryption using `chain.Crypto()`
- HMAC signing to prevent tampering
- Integrates with Chain's Keyring for key rotation

### Custom Store Example

```go
type RedisStore struct {
    client *redis.Client
}

func (s *RedisStore) Name() string {
    return "redis"
}

func (s *RedisStore) Init(config session.Config, router *chain.Router) error {
    s.client = redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    return nil
}

func (s *RedisStore) Get(ctx *chain.Context, sid string) (string, map[string]any) {
    data, err := s.client.Get(ctx.Request.Context(), sid).Bytes()
    if err != nil {
        return "", nil
    }
    // deserialize data
    return sid, deserialize(data)
}

func (s *RedisStore) Put(ctx *chain.Context, sid string, data map[string]any) (string, error) {
    if sid == "" {
        sid = generateSessionID()
    }
    // serialize and store
    return sid, s.client.Set(ctx.Request.Context(), sid, serialize(data), 0).Err()
}

func (s *RedisStore) Delete(ctx *chain.Context, sid string) {
    s.client.Del(ctx.Request.Context(), sid)
}
```

## Security Considerations

### Secret Key Base

Always set a strong secret key base before using sessions:

```go
chain.SetSecretKeyBase(os.Getenv("SECRET_KEY_BASE"))
```

The secret key base should be:
- At least 32 characters long
- Kept secret (never commit to version control)
- Consistent across application restarts (or sessions will be invalidated)

### HTTPS Only

In production, always set `Secure: true` to ensure cookies are only sent over HTTPS:

```go
session.Config{
    Secure: true,
}
```

### HTTP Only

Always set `HttpOnly: true` to prevent JavaScript access and mitigate XSS attacks:

```go
session.Config{
    HttpOnly: true,
}
```

### SameSite Policy

Use `SameSiteStrictMode` for maximum CSRF protection:

```go
session.Config{
    SameSite: http.SameSiteStrictMode,
}
```

### Session Fixation Prevention

Always renew session IDs after authentication:

```go
// After successful login
sess, _ := session.Fetch(ctx)
sess.Put("authenticated", true)
sess.Renew() // Generate new session ID
```

### Key Rotation

The session middleware integrates with Chain's Keyring for automatic key rotation:

```go
// Setup keyring
keyring := chain.NewKeyring("salt", 216000, 32, "sha256")

// Old sessions will still work with previous keys
// New sessions will use current key
```

## Error Handling

```go
sess, err := session.Fetch(ctx)
if err != nil {
    if err == session.ErrCannotFetch {
        // No session manager configured for this context
        ctx.Error("session not configured", 500)
        return err
    }
}
```

## Middleware Registration

### Global Registration

```go
router.Use("/*", sessionManager)
```

### Path-scoped Registration

```go
router.Use("/api/*", sessionManager)
```

### Group-scoped Registration

```go
api := router.Group("/api")
api.Use(sessionManager)
```

## Examples

### Authentication Flow

```go
router.POST("/login", func(ctx *chain.Context) error {
    var loginReq struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    if err := ctx.BindJSON(&loginReq); err != nil {
        return err
    }
    
    // Authenticate user
    if !authenticate(loginReq.Username, loginReq.Password) {
        ctx.Unauthorized(map[string]string{"error": "invalid credentials"})
        return nil
    }
    
    // Create session
    sess, err := session.Fetch(ctx)
    if err != nil {
        return err
    }
    sess.Put("user_id", loginReq.Username)
    sess.Put("login_time", time.Now().Unix())
    sess.Renew() // Prevent session fixation
    
    ctx.OK(map[string]string{"status": "logged in"})
    return nil
})

router.GET("/me", func(ctx *chain.Context) error {
    sess, err := session.Fetch(ctx)
    if err != nil {
        return err
    }
    
    userID := sess.Get("user_id")
    if userID == nil {
        ctx.Unauthorized()
        return nil
    }
    
    ctx.Json(map[string]any{"user_id": userID})
    return nil
})

router.POST("/logout", func(ctx *chain.Context) error {
    sess, err := session.Fetch(ctx)
    if err != nil {
        return err
    }
    sess.Destroy()
    ctx.OK("logged out")
    return nil
})
```

### Shopping Cart

```go
router.POST("/cart/add", func(ctx *chain.Context) error {
    sess, err := session.Fetch(ctx)
    if err != nil {
        return err
    }
    
    var item struct {
        ID    string `json:"id"`
        Qty   int    `json:"qty"`
    }
    if err := ctx.BindJSON(&item); err != nil {
        return err
    }
    
    cart := sess.Get("cart")
    if cart == nil {
        cart = []map[string]any{}
    }
    
    // Add item to cart
    cartSlice := cart.([]map[string]any)
    cartSlice = append(cartSlice, map[string]any{
        "id":  item.ID,
        "qty": item.Qty,
    })
    sess.Put("cart", cartSlice)
    
    ctx.OK(map[string]int{"cart_size": len(cartSlice)})
    return nil
})
```

## API Reference

### Functions

```go
func Fetch(ctx *chain.Context) (*Session, error)
func FetchByKey(ctx *chain.Context, key string) (*Session, error)
```

### Types

```go
type Manager struct {
    Config
    Store Store
}

type Config struct {
    Key      string
    Path     string
    Domain   string
    MaxAge   int
    Secure   bool
    HttpOnly bool
    SameSite http.SameSite
}

type Session struct {
    // Internal state, not directly accessed
}

type Store interface {
    Name() string
    Init(config Config, router *chain.Router) error
    Get(ctx *chain.Context, rawCookie string) (sid string, data map[string]any)
    Put(ctx *chain.Context, sid string, data map[string]any) (rawCookie string, err error)
    Delete(ctx *chain.Context, sid string)
}
```

## Troubleshooting

### Session not persisting

- Ensure `chain.SetSecretKeyBase()` is called before session middleware
- Check that middleware is registered before route handlers
- Verify cookie is being set in browser (check Developer Tools)

### Session data lost on refresh

- Check that session is being fetched correctly
- Ensure `sess.Put()` is called before handler returns
- Verify encryption key hasn't changed between requests

### "cannot fetch session" error

- Session manager not configured for the current context
- Use `router.Use("/*", sessionManager)` for global configuration
- Or use multiple managers with `router.Use("/path/*", manager)`

## See Also

- [Chain Crypto Documentation](../../docs/CRYPTO.md)
- [Chain Context API](../../docs/03-api-reference.md)
- [Security Guidelines](../../docs/05-security-guidelines.md)
