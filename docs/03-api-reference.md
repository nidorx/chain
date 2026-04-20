# Chain Framework - API Reference

**Version:** 1.0.0 (Draft)  
**Last Updated:** April 18, 2026

---

## Table of Contents

1. [Router](#router)
2. [Context](#context)
3. [Request Handling](#request-handling)
4. [Response Writing](#response-writing)
5. [Data Binding](#data-binding)
6. [Middleware](#middleware)
7. [Route Groups](#route-groups)
8. [Cryptography](#cryptography)
9. [Utilities](#utilities)
10. [Request Timeouts](#request-timeouts)
11. [Graceful Shutdown](#graceful-shutdown)

---

## Router

The Router is the core component of Chain. It handles HTTP requests, routes them to the appropriate handlers, and manages the request lifecycle.

### Creating a Router

```go
router := chain.New()
```

### Router Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `HandleOPTIONS` | `bool` | `true` | Automatically handle OPTIONS requests |
| `RedirectFixedPath` | `bool` | `true` | Redirect to clean paths (remove `../`, `//`) |
| `RedirectTrailingSlash` | `bool` | `true` | Redirect `/foo/` to `/foo` and vice versa |
| `HandleMethodNotAllowed` | `bool` | `true` | Return 405 if route exists with different method |
| `PanicHandler` | `func(http.ResponseWriter, *http.Request, any)` | `nil` | Custom panic handler |
| `ErrorHandler` | `func(*Context, error)` | `nil` | Custom error handler |
| `NotFoundHandler` | `http.Handler` | `nil` | Custom 404 handler |
| `GlobalOPTIONSHandler` | `http.Handler` | `nil` | Custom global OPTIONS handler |
| `MethodNotAllowedHandler` | `http.Handler` | `nil` | Custom 405 handler |
| `ReqContext` | `func(*Context) context.Context` | `nil` | Custom request context modifier |

### HTTP Method Handlers

```go
router.GET(path string, handler any) error
router.HEAD(path string, handler any) error
router.OPTIONS(path string, handler any) error
router.POST(path string, handler any) error
router.PUT(path string, handler any) error
router.PATCH(path string, handler any) error
router.DELETE(path string, handler any) error
```

#### Handler Signatures Supported

```go
// Chain handler with error
func(ctx *chain.Context) error

// Chain handler without error
func(ctx *chain.Context)

// Standard http.Handler
http.Handler

// Standard http.HandlerFunc
http.HandlerFunc

// Standard handler function
func(w http.ResponseWriter, r *http.Request)

// Standard handler with error
func(w http.ResponseWriter, r *http.Request) error
```

#### Example

```go
router.GET("/users/:id", func(ctx *chain.Context) error {
    id := ctx.GetParam("id")
    ctx.Json(map[string]string{"id": id})
    return nil
})
```

### Generic Handle Method

```go
router.Handle(method string, path string, handler any) error
```

#### Example

```go
router.Handle("GET", "/users", func(ctx *chain.Context) error {
    // handler code
    return nil
})
```

---

## Context

The Context object encapsulates the request and response, providing methods to access parameters, headers, body, and write responses.

### Accessing Route Parameters

```go
// Get parameter by name
value := ctx.GetParam("name")

// Get parameter by index
value := ctx.GetParamByIndex(0)
```

### Query Parameters

```go
// Get query parameter with optional default
value := ctx.QueryParam("page", "1")

// Get query parameter as integer
page := ctx.QueryParamInt("page", 1)
```

### Request Information

```go
host := ctx.Host()        // Request host
ip := ctx.Ip()           // Client IP
method := ctx.Method()   // HTTP method
userAgent := ctx.UserAgent() // User agent
url := ctx.URL()         // Full URL
contentType := ctx.GetContentType() // Content-Type header
```

### Headers

```go
// Get request header
value := ctx.GetHeader("Authorization")

// Get cookie
cookie := ctx.GetCookie("session_id")
```

### Request Body

```go
// Get body as bytes
body, err := ctx.BodyBytes()
```

### Context Data Storage

```go
// Store data in context
ctx.Set("user", userObject)

// Retrieve data from context
if user, exists := ctx.Get("user"); exists {
    // use user
}
```

### Child Contexts

```go
// Create child context
child := ctx.Child()

// Create child with custom parameters
child := ctx.WithParams([]string{"id"}, []string{"123"})

// Destroy child context
child.Destroy()
```

---

## Response Writing

### Status Codes

All status methods accept optional content (string, []byte, or any for JSON).

```go
// Basic status code
ctx.Status(http.StatusAccepted)             // 202
ctx.WriteHeader(http.StatusOK)              // 200 (alias)

// Status with text content
ctx.Status(200, "OK")                       // 200 with text
ctx.Status(404, "Not Found")                // 404 with text

// Status with binary content
ctx.Status(200, []byte{...})                // 200 with bytes

// Status with JSON content
ctx.Status(200, map[string]string{...})     // 200 with JSON
ctx.Status(201, struct{...})                // 201 with JSON

// Convenience methods (all accept optional content)
ctx.OK()                                    // 200 OK
ctx.OK("success")                           // 200 with text
ctx.OK([]byte{...})                         // 200 with bytes
ctx.OK(map[string]string{"msg": "ok"})      // 200 with JSON

ctx.Created()                               // 201 Created
ctx.Created("resource created")             // 201 with text
ctx.Created(map[string]string{"id": "123"}) // 201 with JSON

ctx.NoContent()                             // 204 No Content (no content accepted)

ctx.BadRequest()                            // 400 Bad Request
ctx.BadRequest("invalid input")             // 400 with custom message
ctx.BadRequest(map[string]string{"error": "validation failed"}) // 400 with JSON

ctx.Unauthorized()                          // 401 Unauthorized
ctx.Unauthorized("login required")          // 401 with text
ctx.Unauthorized(map[string]string{"error": "invalid token"})   // 401 with JSON

ctx.Forbidden()                             // 403 Forbidden
ctx.Forbidden("access denied")              // 403 with text
ctx.Forbidden(map[string]string{"error": "insufficient permissions"}) // 403 with JSON

ctx.NotFound()                              // 404 Not Found
ctx.NotFound("resource missing")            // 404 with text
ctx.NotFound(map[string]string{"error": "not found"}) // 404 with JSON

ctx.TooManyRequests()                       // 429 Too Many Requests
ctx.TooManyRequests("rate limit exceeded")  // 429 with text
ctx.TooManyRequests(map[string]any{"retry_after": 60}) // 429 with JSON

ctx.InternalServerError()                   // 500 Internal Server Error
ctx.InternalServerError("server error")     // 500 with text
ctx.InternalServerError(map[string]string{"error": "internal error"}) // 500 with JSON

ctx.NotImplemented()                        // 501 Not Implemented
ctx.NotImplemented("feature not available") // 501 with text
ctx.NotImplemented(map[string]string{"error": "not implemented"}) // 501 with JSON

ctx.ServiceUnavailable()                    // 503 Service Unavailable
ctx.ServiceUnavailable("maintenance")       // 503 with text
ctx.ServiceUnavailable(map[string]string{"error": "service unavailable"}) // 503 with JSON
```

#### Content Type Behavior

When content is provided to status methods, the response content type is automatically determined:

- **string**: Sent as plain text
- **[]byte**: Sent as raw bytes
- **any other type**: Encoded as JSON (Content-Type: application/json)

#### Error Method

The `Error` method provides specialized error handling with optional custom content:

```go
// Basic error response (plain text)
ctx.Error("Error message", http.StatusBadRequest)

// Error with custom text message
ctx.Error("ignored", http.StatusBadRequest, "custom error message")

// Error with JSON response (bypasses http.Error)
ctx.Error("ignored", http.StatusBadRequest, map[string]string{"error": "validation failed"})
```

### JSON Response

```go
ctx.Json(map[string]string{
    "message": "Hello, World!",
})
```

### Generic Response

```go
// Set header
ctx.SetHeader("Content-Type", "application/json")

// Add header
ctx.AddHeader("X-Custom-Header", "value")

// Write data
ctx.Write([]byte("Hello"))

// Serve content with ETag and caching
ctx.ServeContent(data, "filename.txt", modTime)
```

### Redirect

```go
ctx.Redirect("/new-path", http.StatusMovedPermanently)
```

### Cookies

```go
// Set cookie
ctx.SetCookie(&http.Cookie{
    Name:     "session",
    Value:    "abc123",
    Path:     "/",
    HttpOnly: true,
    MaxAge:   3600,
})

// Remove cookie
ctx.RemoveCookie("session")
```

### Response Hooks

```go
// Before response is sent
ctx.BeforeSend(func() {
    // modify headers, log, etc.
})

// After response is sent
ctx.AfterSend(func() {
    // cleanup, metrics, etc.
})
```

---

## Data Binding

Chain provides automatic data binding from requests to Go structs.

### Automatic Binding

```go
type User struct {
    Name  string `json:"name" query:"name"`
    Email string `json:"email" query:"email"`
}

var user User
if err := ctx.Bind(&user); err != nil {
    ctx.BadRequest()
    return err
}
```

### Binding Sources

| Method | Source |
|--------|--------|
| `Bind()` / `ShouldBind()` | Auto-detect (query, path, header, body) |
| `BindJSON()` / `ShouldBindJSON()` | JSON body |
| `BindXML()` / `ShouldBindXML()` | XML body |
| `BindForm()` / `ShouldBindForm()` | Form data |
| `BindQuery()` / `ShouldBindQuery()` | Query parameters |
| `BindPath()` / `ShouldBindPath()` | Path parameters |
| `BindHeader()` / `ShouldBindHeader()` | Request headers |
| `BindFormPost()` / `ShouldBindFormPost()` | POST form data |
| `BindFormMultipart()` / `ShouldBindFormMultipart()` | Multipart form |

### Bind vs ShouldBind

- `Bind()` - Automatically returns 400 error if binding fails
- `ShouldBind()` - Returns error without setting status code (you handle it)

### Validation

Chain integrates with `go-playground/validator/v10`.

```go
type User struct {
    Name  string `json:"name" binding:"required,min=3,max=50"`
    Email string `json:"email" binding:"required,email"`
    Age   int    `json:"age" binding:"min=0,max=150"`
}

var user User
if err := ctx.Bind(&user); err != nil {
    // Validation error
    ctx.BadRequest()
    return err
}
```

---

## Middleware

Middleware functions intercept requests before they reach handlers and can modify requests/responses.

### Registering Middleware

```go
// Global middleware (all routes)
router.Use(func(ctx *chain.Context, next func() error) error {
    // Before handler
    log.Println("Request:", ctx.Method(), ctx.path)
    
    err := next()
    
    // After handler
    log.Println("Response:", ctx.GetStatus())
    
    return err
})

// Route-specific middleware
router.Use("/api/*", authMiddleware)
router.Use("GET", "/admin/*", adminMiddleware)
```

### Middleware Signatures Supported

```go
// Simple function
func()

// With error
func() error

// With context
func(ctx *Context)
func(ctx *Context) error

// With next function
func(next func() error)
func(next func() error) error
func(ctx *Context, next func() error)
func(ctx *Context, next func() error) error

// Standard http.Handler
http.Handler
http.HandlerFunc

// Interface-based
MiddlewareHandler
MiddlewareWithInitHandler
```

### Middleware Examples

#### Logging Middleware

```go
router.Use(func(ctx *chain.Context, next func() error) error {
    start := time.Now()
    
    err := next()
    
    duration := time.Since(start)
    log.Printf("[%d] %s %s (%v)", 
        ctx.GetStatus(), 
        ctx.Method(), 
        ctx.path, 
        duration)
    
    return err
})
```

#### Authentication Middleware

```go
router.Use("/api/*", func(ctx *chain.Context, next func() error) error {
    token := ctx.GetHeader("Authorization")
    if token == "" {
        ctx.Unauthorized()
        return nil // Don't call next()
    }
    
    // Validate token
    if !isValidToken(token) {
        ctx.Unauthorized()
        return nil
    }
    
    return next()
})
```


## Route Groups

Route groups allow organizing routes under a common prefix.

### Creating Groups

```go
// Create API group
api := router.Group("/api")

// Create versioned group
v1 := api.Group("/v1")

// Register routes
v1.GET("/users", getUsersHandler)
v1.POST("/users", createUserHandler)
v1.GET("/users/:id", getUserHandler)
```

### Nested Groups

```go
v1 := router.Group("/api").Group("/v1")
v1.GET("/users", handler) // /api/v1/users
```

### Group Middleware

```go
api := router.Group("/api")
api.Use(authMiddleware) // Only applies to /api/* routes

api.GET("/public", publicHandler)      // Has auth
api.GET("/private", privateHandler)    // Has auth
```

### Group Handle Method

```go
api.Handle("GET", "/custom", handler)
```

---

## Cryptography

Chain provides comprehensive cryptographic utilities for encryption, decryption, and message signing.

### Setup

```go
// Set secret key (required before using crypto functions)
chain.SetSecretKeyBase("your-secret-key-32-bytes-long!!")
```

### Encryption/Decryption

```go
// Encrypt data
data := []byte("Secret message")
aad := []byte("additional data") // Optional
encrypted, err := chain.Crypto().Encrypt(secretKey, data, aad)

// Decrypt data
decrypted, err := chain.Crypto().Decrypt(secretKey, encrypted, aad)
```

### Message Signing

```go
// Sign a message
signature := chain.Crypto().MessageSign(secretKey, message, "sha256")

// Verify a signature
decoded, err := chain.Crypto().MessageVerify(secretKey, signedMessage)
```

### Message Encryption (Authenticated)

```go
// Encrypt with authentication
encoded, err := chain.Crypto().MessageEncrypt(secretKey, content, aad)

// Decrypt with authentication
content, err := chain.Crypto().MessageDecrypt(secretKey, encoded, aad)
```

### Key Derivation

```go
// Derive a key using PBKDF2
derivedKey := chain.Crypto().KeyGenerate(
    secret,      // Secret key
    salt,        // Salt
    216000,      // Iterations (minimum recommended)
    32,          // Key length
    "sha256",    // Hash function
)
```

### Keyring (Key Rotation)

```go
// Create a keyring that auto-updates with SecretKeyBase changes
keyring := chain.NewKeyring("salt", 216000, 32, "sha256")

// Encrypt with primary key
encrypted, err := keyring.Encrypt(data, aad)

// Decrypt (tries all keys)
decrypted, err := keyring.Decrypt(encrypted, aad)

// Sign with primary key
signature, err := keyring.MessageSign(message, "sha256")

// Verify (tries all keys)
decoded, err := keyring.MessageVerify(signedMessage)
```

---

## Utilities

### Unique ID Generation

```go
// Generate a KSUID (K-Sortable Unique IDentifier)
uid := ctx.NewUID()
// or
uid := chain.NewUID()
```

### Hashing Functions

```go
// SHA-256 based (via xxhash)
etag := chain.HashXxh64(content)

// CRC32
checksum := chain.HashCrc32(content)

// MD5 (deprecated - use for non-security purposes only)
hash := chain.HashMD5(text)
```

### Serializer Interface

```go
type Serializer interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) (any, error)
}

// JSON serializer
jsonSerializer := &chain.JsonSerializer{}
data, _ := jsonSerializer.Encode(object)
object, _ = jsonSerializer.Decode(data, &object)
```

---

## Request Timeouts

Chain provides request timeout enforcement at both global and per-route levels. When a timeout expires, the handler's response writing is blocked and a `503 Service Unavailable` is returned.

### Timeout Middleware (Global)

Applies a timeout to all routes:

```go
import "github.com/nidorx/chain"

// 30-second timeout for all requests
router.Use(chain.TimeoutMiddleware(30 * time.Second))
```

### WithTimeout (Per-Route)

Applies a timeout to a specific handler:

```go
router.GET("/slow", chain.WithTimeout(10*time.Second, func(ctx *chain.Context) error {
    // This handler has 10 seconds to complete
    result := doSlowOperation()
    ctx.Json(result)
    return nil
}))
```

### WithTimeoutMiddleware (Path-Scoped)

Applies a timeout to a path pattern:

```go
// 15-second timeout for all /api/* routes
router.Use("/api/*", chain.WithTimeoutMiddleware(15*time.Second))

// 5-second timeout for health checks
router.Use("/health", chain.WithTimeoutMiddleware(5*time.Second))
```

### Behavior

| Scenario | Result |
|----------|--------|
| Handler completes before timeout | Normal response |
| Handler exceeds timeout | `503 Service Unavailable`, handler blocked from writing |
| Handler already wrote response before timeout | Response sent as-is |
| Zero or negative timeout | No timeout enforced (passthrough) |

### Error Value

When a timeout occurs, the middleware returns `chain.ErrRequestTimeout`:

```go
router.ErrorHandler = func(ctx *chain.Context, err error) {
    if errors.Is(err, chain.ErrRequestTimeout) {
        log.Printf("Request timed out: %s %s", ctx.Method(), ctx.Request.URL.Path)
    }
}
```

### Timeout Configuration

| Type | Description |
|------|-------------|
| `TimeoutMiddleware(duration)` | Global middleware, applies to all routes |
| `WithTimeout(duration, handler)` | Wraps a single handler with timeout |
| `WithTimeoutMiddleware(duration)` | MiddlewareHandler for path-scoped timeouts |

---

## Graceful Shutdown

Chain provides a `Server` type that wraps `http.Server` with built-in graceful shutdown support, handling OS signals and draining in-flight requests.

### Basic Usage

```go
import "github.com/nidorx/chain"

func main() {
    r := chain.New()
    r.GET("/", func(ctx *chain.Context) error {
        ctx.OK()
        return nil
    })

    server := chain.NewServer(r, ":8080")
    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

### Server Creation

```go
// Simple creation
server := chain.NewServer(router, ":8080")

// With custom configuration
server := chain.NewServerWithConfig(router, ":8080", chain.ShutdownConfig{
    Timeout: 60 * time.Second,
    Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP},
})
```

### Shutdown Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Timeout` | `time.Duration` | `30s` | Max duration to wait for in-flight requests |
| `Signals` | `[]os.Signal` | `SIGINT, SIGTERM` | OS signals that trigger shutdown |

### Lifecycle Hooks

```go
server := chain.NewServer(router, ":8080")

// Called when shutdown begins (before waiting for in-flight requests)
server.OnShutdown(func() {
    log.Println("Shutdown initiated, stopping new connections...")
})

// Called after all in-flight requests complete or timeout reached
server.OnStop(func() {
    log.Println("Server stopped cleanly")
})
```

### Programmatic Shutdown

```go
// Shutdown from application code
server := chain.NewServer(router, ":8080")

// Start server in background
go func() {
    if err := server.ListenAndServe(); err != nil {
        log.Printf("Server error: %v", err)
    }
}()

// Later: trigger shutdown
time.AfterFunc(1*time.Hour, func() {
    server.Stop() // or server.Shutdown(context.Background())
})
```

### Shutdown Methods

| Method | Description |
|--------|-------------|
| `ListenAndServe()` | Start server with signal-based graceful shutdown |
| `ListenAndServeTLS(cert, key)` | Start TLS server with signal-based graceful shutdown |
| `Shutdown(ctx)` | Initiate shutdown with custom context for timeout |
| `Stop()` | Convenience for `Shutdown(nil)` (uses configured timeout) |
| `IsShuttingDown()` | Returns true if shutdown is in progress |
| `Wait()` | Blocks until server has fully shut down |
| `OnShutdown(fn)` | Register callback at shutdown start |
| `OnStop(fn)` | Register callback at shutdown completion |

### Graceful Middleware

During shutdown, you may want to signal clients that connections will close:

```go
server := chain.NewServer(router, ":8080")

// Add middleware that sets Connection: close during shutdown
router.Use(chain.GracefulMiddleware(server))
```

This sets the `Connection: close` header on all responses while the server is shutting down, preventing keep-alive connections from persisting.

### Shutdown Sequence

```
1. OS signal received (SIGINT/SIGTERM)
   └─2. OnShutdown() callback invoked
       └─3. http.Server.Shutdown() called
           ├─4. Stop accepting new connections
           ├─5. Wait for in-flight requests (up to Timeout)
           └─6. OnStop() callback invoked
               └─7. Wait() channel closed
```

### Complete Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/nidorx/chain"
)

func main() {
    r := chain.New()
    r.GET("/", func(ctx *chain.Context) error {
        ctx.Json(map[string]string{"status": "ok"})
        return nil
    })

    server := chain.NewServerWithConfig(r, ":8080", chain.ShutdownConfig{
        Timeout: 30 * time.Second,
    })

    server.OnShutdown(func() {
        log.Println("Shutting down server...")
    })
    server.OnStop(func() {
        log.Println("Server stopped")
    })

    // Graceful middleware - sets Connection: close during shutdown
    r.Use(chain.GracefulMiddleware(server))

    if err := server.ListenAndServe(); err != nil {
        log.Printf("Server error: %v", err)
        os.Exit(1)
    }
}
```

---

## Error Handling

### Custom Error Handler

```go
router.ErrorHandler = func(ctx *chain.Context, err error) {
    log.Printf("Error: %v", err)
    ctx.Json(map[string]string{
        "error": "Internal Server Error",
    })
    ctx.StatusInternalServerError()
}
```

### Custom Panic Handler

```go
router.PanicHandler = func(w http.ResponseWriter, r *http.Request, rcv any) {
    log.Printf("Panic: %v", rcv)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
```

### Custom Not Found Handler

```go
router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte(`{"error":"Not Found"}`))
})
```

---

## Route Patterns

### Static Routes

```go
router.GET("/users", handler)
```

### Parameterized Routes

```go
router.GET("/users/:id", handler)
router.GET("/users/:user_id/posts/:post_id", handler)
```

### Wildcard Routes

```go
router.GET("/files/*filepath", handler)
// Matches: /files/js/app.js, /files/css/style.css
```

### Mixed Routes

```go
router.GET("/api/:version/users/:id/*path", handler)
// Matches: /api/v1/users/123/profile/photo
```

---

## Route Lookup

For programmatic route lookup (useful for testing or internal routing):

```go
route, ctx := router.Lookup("GET", "/users/123")
if route != nil {
    route.Dispatch(ctx)
}
```

---

## PubSub

Chain includes a cluster-aware publish/subscribe system for real-time message distribution.

### Subscribing to Topics

```go
import "github.com/nidorx/chain/pubsub"

// Create a dispatcher
type MyDispatcher struct{}

func (d *MyDispatcher) Dispatch(topic string, message []byte, from string) {
    log.Printf("Message on %s from %s: %s", topic, from, message)
}

// Subscribe with topic pattern (supports wildcards)
dispatcher := &MyDispatcher{}
pubsub.Subscribe("user:*", dispatcher)
pubsub.Subscribe("notifications", dispatcher)
```

### Broadcasting Messages

```go
// Broadcast to all nodes in the cluster
err := pubsub.Broadcast("user:123", []byte(`{"event":"update"}`))

// Broadcast only on the current node
pubsub.LocalBroadcast("user:123", []byte(`{"event":"local"}`))

// Direct broadcast to a specific node
pubsub.DirectBroadcast("node-ksuid", "user:123", []byte(`{"event":"direct"}`))
```

### Topic Patterns

Topics support wildcard patterns:

```go
pubsub.Subscribe("user:*", dispatcher)       // Matches user:123, user:456
pubsub.Subscribe("user:123:*", dispatcher)   // Matches user:123:profile, user:123:settings
pubsub.Subscribe("*", dispatcher)            // Matches all topics
```

### Adapter Configuration

For distributed pub/sub, configure an adapter (e.g., Redis, NATS):

```go
type Adapter interface {
    Name() string
    Subscribe(topic string)
    Unsubscribe(topic string)
    Broadcast(topic string, message []byte, opts map[string]any) error
}

// Set adapters for specific topics
pubsub.SetAdapters([]pubsub.AdapterConfig{
    {
        Adapter:  myRedisAdapter,
        Topics:   []string{"user:*", "order:*"},
    },
})
```

### Options

```go
// Set global broadcast options
pubsub.SetGlobalOptions(
    pubsub.O("encrypted", true),
    pubsub.O("compressed", true),
)

// Per-message options
pubsub.Broadcast("topic", data, pubsub.O("encrypted", false))
```

### Node Identification

```go
// Get current node's KSUID
nodeID := pubsub.Self()
```

---

## Socket & Channels

Real-time multiplexed communication over Server-Sent Events (SSE).

### Handler Configuration

```go
import "github.com/nidorx/chain/socket"

var AppSocket = &socket.Handler{
    Channels: []*socket.Channel{
        socket.NewChannel("chat:*", chatChannel),
        socket.NewChannel("presence:*", presenceChannel),
    },
    Serializer:  &socket.MessageSerializer{},
    OnConnect: func(session *socket.Session) error {
        log.Printf("New session: %s", session.Id())
        return nil
    },
}
```

### Configure on Router

```go
router := chain.New()
router.Configure("/socket", AppSocket)
// SSE endpoint available at /socket
```

### Channel Definition

```go
func chatChannel(channel *socket.Channel) {
    // Handle joins
    channel.Join("chat:lobby", func(params any, socket *socket.Socket) (reply any, err error) {
        log.Printf("User joined: %s", socket.Id())
        socket.Push("welcome", map[string]string{"message": "Welcome to the lobby!"})
        return nil, nil
    })

    // Handle incoming events
    channel.HandleIn("send_message", func(event string, payload any, socket *socket.Socket) (reply any, err error) {
        msg := payload.(map[string]any)
        // Broadcast to all subscribers
        channel.Broadcast("chat:lobby", "new_message", map[string]any{
            "user":    socket.Id(),
            "message": msg["text"],
            "time":    time.Now().Format(time.RFC3339),
        })
        return map[string]string{"status": "ok"}, nil
    })

    // Handle outgoing events (intercept/modify)
    channel.HandleOut("new_message", func(event string, payload any, socket *socket.Socket) {
        log.Printf("Outgoing message to %s: %v", socket.Id(), payload)
    })

    // Handle leave
    channel.Leave("chat:lobby", func(socket *socket.Socket, reason socket.LeaveReason) {
        log.Printf("User left: %s (reason: %d)", socket.Id(), reason)
    })
}
```

### Broadcasting

```go
// Broadcast to all sockets in the cluster
channel.Broadcast("chat:lobby", "announcement", map[string]string{
    "message": "Server maintenance in 5 minutes",
})

// Broadcast only on the current node
channel.LocalBroadcast("chat:lobby", "local_event", nil)
```

### Subscribe to PubSub Topics

Channels can auto-push pub/sub messages to connected clients:

```go
// Subscribe to a pubsub topic and push to clients
channel.Subscribe("chat:lobby:events", "new_event")
// When pubsub receives a message on "chat:lobby:events",
// it pushes "new_event" to all connected clients
```

### Session Management

```go
// Get session by ID
session := AppSocket.GetSession("session-id")

// Resume an existing session (for reconnection)
session := AppSocket.Resume("session-id")

// Create a new session programmatically
session, err := AppSocket.Connect("/socket", map[string]string{"user_id": "123"})
```

### Session API

```go
session.Id()                     // Session ID
session.Endpoint()               // Socket endpoint path
session.Closed()                 // Whether session is closed
session.Push([]byte{...})        // Push raw bytes to client
session.Dispatch([]byte{...})    // Dispatch message to channel
session.ScheduleShutdown(30 * time.Second) // Schedule disconnect
session.StopScheduledShutdown()  // Cancel scheduled disconnect
```

### Socket API

```go
socket.Id()              // Socket ID
socket.Topic()           // Current topic
socket.Status()          // Joining, Joined, Leaving, Removed
socket.Session()         // Parent session
socket.Get("key")        // Get server-side data
socket.Set("key", value) // Set server-side data
socket.Push("event", payload) // Push event to client
socket.Broadcast("event", payload) // Broadcast to topic subscribers
```

### Leave Reasons

```go
socket.LeaveReasonLeave   // Client initiated leave
socket.LeaveReasonRejoin  // Socket is rejoining
socket.LeaveReasonClose   // Session closing
```

---

## Session Middleware

Cookie-based sessions with optional encryption and signing.

### Fetching Sessions

```go
import "github.com/nidorx/chain/middlewares/session"

// Fetch session from context
sess, err := session.Fetch(ctx)
if err != nil {
    // No session found
}

// Fetch by specific key
sess, err := session.FetchByKey(ctx, "custom_key")
```

### Session Operations

```go
// Store values
sess.Put("user_id", "123")
sess.Put("role", "admin")

// Retrieve values
userID := sess.Get("user_id") // returns any
role := sess.Get("role").(string)

// Check existence
if sess.Exist("user_id") {
    // key exists
}

// Delete
sess.Delete("temp_data")

// Clear all
sess.Clear()

// Renew session ID (security best practice after login)
sess.Renew()

// Destroy session (logout)
sess.Destroy()

// Ignore changes made during this request
sess.IgnoreChanges()

// Get all data
data := sess.GetMap() // returns map[string]any
```

### Cookie Store Configuration

```go
store := &session.Cookie{
    SigningKeyring:    signingKeyring,    // for HMAC signing
    EncryptionKeyring: encryptionKeyring, // for AES-GCM encryption
    EncryptionAAD:     []byte("session-aad"),
    Serializer:        &chain.JsonSerializer{},
}

manager := &session.Manager{
    Store: store,
    Config: session.Config{
        Key:      "chain_session",
        Path:     "/",
        MaxAge:   86400, // 24 hours
        Secure:   true,
        HttpOnly: true,
        SameSite: http.SameSiteStrictMode,
    },
}
```

---

## Node Management

### Node Identity

```go
// Set the node name (must be in "name@host" format)
err := chain.SetNodeName("app-server-1@localhost")

// Get current node name
name := chain.NodeName()
```

Node names are used by the pub/sub system for node identification.

---

## Secret Key Management

### Setting the Secret Key

```go
// Set at application startup
err := chain.SetSecretKeyBase(os.Getenv("SECRET_KEY_BASE"))
if err != nil {
    log.Fatalf("Invalid secret key: %v", err)
}
```

### Reading the Secret Key

```go
// Get current secret key base
key := chain.SecretKeyBase()

// Get all keys in rotation (for keyring sync)
keys := chain.SecretKeys()
```

### Secret Key Sync

```go
// Register a callback that fires when the secret key changes
cancel := chain.SecretKeySync(func(key string) {
    // Update external systems with new key
    updateRedisAuth(key)
})

// Cancel the sync callback when no longer needed
cancel()
```

---

## ResponseWriterSpy

Chain wraps `http.ResponseWriter` with `ResponseWriterSpy` to track write state.

```go
type ResponseWriterSpy struct {
    http.ResponseWriter
    // ... internal fields
}

// Access from context
writer := ctx.Writer.(*chain.ResponseWriterSpy)

status := writer.Status()       // Get response status code
started := ctx.WriteStarted()   // Has response been started?
written := ctx.WriteCalled()    // Was Write() called?
hdrWritten := ctx.WriteHeaderCalled() // Was WriteHeader() called?
```

### Error

```go
chain.ErrAlreadySent // Error when trying to modify/send an already sent response
```

---

## Middleware Helpers

### MiddlewareFunc

```go
type MiddlewareFunc func(ctx *chain.Context, next func() error) error
```

### MiddlewareChain

```go
// Create a chain of middlewares
mc := chain.NewMiddlewareChain(
    loggingMiddleware,
    authMiddleware,
    corsMiddleware,
)

// Add more middlewares
mc.Add(extraMiddleware)

// Execute the chain
err := mc.Execute(ctx, handler)
```

### WrapHandler

Wrap a standard `http.Handler` as a Chain middleware:

```go
// Wrap net/http middleware
chainMW := chain.WrapHandler(someHttpHandler)

// Use with Chain router
router.Use(chainMW)
```

### MaxBytesMiddleware

Limit request body size:

```go
// Limit to 10MB
router.Use(chain.MaxBytesMiddleware(10 << 20))
```

### ContentTypeMiddleware

Restrict allowed Content-Types:

```go
// Only allow JSON and XML
router.Use(chain.ContentTypeMiddleware("application/json", "application/xml"))
```

---

## Group Interface

Route groups implement the `chain.Group` interface:

```go
type Group interface {
    GET(route string, handle any) error
    HEAD(route string, handle any) error
    OPTIONS(route string, handle any) error
    POST(route string, handle any) error
    PUT(route string, handle any) error
    PATCH(route string, handle any) error
    DELETE(route string, handle any) error
    Use(args ...any) (Group, error)
    Group(route string) Group
    Handle(method string, route string, handle any) error
    Configure(route string, configurator RouteConfigurator)
}
```

### RouterGroup

```go
// Create a group
api := router.Group("/api")

// Nested groups
v1 := api.Group("/v1")

// Add middleware to a group
api.Use(authMiddleware)

// Register routes
api.GET("/users", handler)
```

---

## RouteConfigurator Interface

For custom route configuration:

```go
type RouteConfigurator interface {
    Configure(router *chain.Router, path string)
}

// Usage
router.Configure("/socket", AppSocket) // AppSocket implements RouteConfigurator
```

---

## RouteInfo Accessors

Access parsed route information:

```go
route := ctx.Route

// Path details
path := route.Path()         // Original path pattern (/users/:id)
pattern := route.Pattern()   // Normalized pattern (/users/:)
segments := route.Segments() // ["users", ":id"]

// Parameter details
params := route.Params()         // ["id"]
paramIdx := route.ParamsIndex()  // [1] (positions in segments)

// Route characteristics
hasStatic := route.HasStatic()     // true
hasParam := route.HasParameter()   // true
hasWildcard := route.HasWildcard() // false
priority := route.Priority()       // Calculated priority score

// Replace path params with values
url := route.ReplacePath(ctx) // e.g., /users/123

// Matching
matches := route.FastMatch(ctx) // Fast segment-based match
```

---

## Route Types

```go
// Handle function type
type Handle func(*Context) error

// Route struct
type Route struct {
    Info        *RouteInfo    // Route pattern info
    Handle      Handle        // Handler function
    Middlewares []*Middleware // Applied middlewares
}

// Dispatch route to handler
err := route.Dispatch(ctx)
```

---

## Middleware Types

```go
// Standard middleware interface
type MiddlewareHandler interface {
    Handle(ctx *Context, next func() error) error
}

// Middleware with initialization
type MiddlewareWithInitHandler interface {
    Init(method string, path string, router *Router)
    Handle(ctx *Context, next func() error) error
}

// Middleware struct
type Middleware struct {
    Path   *RouteInfo                              // Path pattern
    Handle func(ctx *Context, next func() error) error // Handler
}
```

---

## Validation Engine

Chain integrates with `go-playground/validator` for struct validation.

### Default Validator

```go
// Access the validator
validator := chain.Validator

// Validate any struct
err := validator.ValidateStruct(&myStruct)

// Access underlying engine (go-playground/validator)
engine := validator.Engine()
```

### Custom Validator

Replace with your own validator implementation:

```go
type MyValidator struct{}

func (v *MyValidator) ValidateStruct(obj any) error {
    // Custom validation logic
    return nil
}

func (v *MyValidator) Engine() any {
    return nil
}

chain.Validator = &MyValidator{}
```

### SliceValidationError

```go
// Error type for multiple validation errors
type SliceValidationError []error

// Usage in error handler
if errs, ok := err.(chain.SliceValidationError); ok {
    for _, e := range errs {
        log.Printf("Validation error: %v", e)
    }
}
```

---

## Binding Configuration

### JSON Decoder Options

```go
// Use json.Number instead of float64 for numbers
chain.EnableDecoderUseNumber = true

// Reject unknown JSON fields
chain.EnableDecoderDisallowUnknownFields = true
```

### BindingDefaultStruct

```go
// Configure default binding behavior
binding := chain.BindingDefault
binding.(*chain.BindingDefaultStruct).BindHeader = true // Include headers in auto-binding
```

### MapFormWithTag

Map form data to struct with a custom tag:

```go
type MyStruct struct {
    Name string `custom:"name"`
}

var s MyStruct
err := chain.MapFormWithTag(&s, formData, "custom")
```

### BindUnmarshaler

Implement custom unmarshaling for form/query values:

```go
type CustomType struct {
    Value int
}

func (c *CustomType) UnmarshalParam(param string) error {
    v, err := strconv.Atoi(param)
    if err != nil {
        return err
    }
    c.Value = v
    return nil
}
```

---

## Error Types

### RouteValidationError

```go
type RouteValidationError struct {
    Field   string // Field name
    Value   string // Invalid value
    Message string // Error description
}

// Create validation error
err := chain.NewRouteValidationError("email", "not-an-email", "must be a valid email")
```

### Validation Functions

```go
// Validate route path syntax
err := chain.ValidateRoutePath("/users/:id")

// Validate route method
err := chain.ValidateRouteMethod("GET")

// Validate route handler
err := chain.ValidateRouteHandler(handler)

// Validate query parameter
err := chain.ValidateQueryParameter("q", value, chain.DefaultMaxQueryParameterLength)

// Validate header value
err := chain.ValidateHeaderValue("Authorization", value, chain.DefaultMaxHeaderLength)

// Validate request body size
err := chain.ValidateRequestBodySize(contentLength, chain.DefaultMaxRequestBodySize)
```

### Sanitization

```go
// Sanitize URL path
cleanPath := chain.SanitizePath("/foo/../bar")

// Sanitize header value
safeValue := chain.SanitizeHeaderValue("value with\r\ninjection")
```

### Default Limits

```go
chain.DefaultMaxQueryParameterLength  // 1024 bytes
chain.DefaultMaxHeaderLength          // 4096 bytes
chain.DefaultMaxRequestBodySize       // 10 MB
```

---

## PubSub Message Types

```go
// Message types for cluster communication
pubsub.MessageTypeCompress
pubsub.MessageTypeEncrypt
pubsub.MessageTypeBroadcast
pubsub.MessageTypeDirectBroadcast
pubsub.IndirectPingMsg
pubsub.AckRespMsg
pubsub.SuspectMsg
pubsub.AliveMsg
pubsub.DeadMsg
pubsub.PushPullMsg
pubsub.CompoundMsg
pubsub.UserMsg
pubsub.NackRespMsg
pubsub.ErrMsg
```

---

## Pkg Utilities

### PathClean

```go
import "github.com/nidorx/chain/pkg"

// Clean URL paths (removes . and .. elements)
clean := pkg.PathClean("/foo/../bar")  // Returns "/bar"
clean = pkg.PathClean("/foo//bar")     // Returns "/foo/bar"
clean = pkg.PathClean("/foo/./bar")    // Returns "/foo/bar"
```

### WildcardStore

Generic wildcard pattern matching store:

```go
store := pkg.NewWildcardStore[string]()

// Insert with wildcard pattern
store.Insert("user:*", "user-handler")
store.Insert("user:123:*", "specific-handler")

// Get by exact key
item := store.Get("user:*") // "user-handler"

// Match by prefix
item = store.Match("user:123:profile") // "user:123:*" (most specific)
item = store.Match("user:456:profile") // "user:*" (fallback)

// Match all patterns
items := store.MatchAll("user:123:profile") // ["user:*", "user:123:*"]
```

---

## Socket Message Types

```go
// Message kinds
socket.MessageTypePush     // Push to client
socket.MessageTypeReply    // Reply to request
socket.MessageTypeBroadcast // Broadcast to topic

// Reply status codes
socket.ReplyStatusCodeOk     = 0
socket.ReplyStatusCodeError  = 1
```

### Message Structure

```go
type Message struct {
    Kind     socket.MessageType
    JoinRef  int
    Ref      int
    Status   int
    Topic    string
    Event    string
    Payload  any
}
```

---

## Complete Example

```go
package main

import (
    "github.com/nidorx/chain"
    "net/http"
)

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

func main() {
    router := chain.New()
    
    // Set secret key for crypto operations
    chain.SetSecretKeyBase("your-32-byte-secret-key!!")
    
    // Global middleware
    router.Use(func(ctx *chain.Context, next func() error) error {
        log.Printf("Request: %s %s", ctx.Method(), ctx.path)
        return next()
    })
    
    // Route groups
    api := router.Group("/api")
    v1 := api.Group("/v1")
    
    // Get all users
    v1.GET("/users", func(ctx *chain.Context) error {
        ctx.Json(map[string]string{
            "users": "list",
        })
        return nil
    })
    
    // Create user
    v1.POST("/users", func(ctx *chain.Context) error {
        var user User
        if err := ctx.BindJSON(&user); err != nil {
            return err
        }
        
        user.ID = ctx.NewUID()
        ctx.Status(http.StatusCreated)
        ctx.Json(user)
        return nil
    })
    
    // Get user by ID
    v1.GET("/users/:id", func(ctx *chain.Context) error {
        id := ctx.GetParam("id")
        
        // Encrypt user ID for secure response
        encrypted, _ := chain.Crypto().MessageEncrypt(
            []byte(chain.SecretKeyBase()), 
            []byte(id), 
            nil,
        )
        
        ctx.Json(map[string]string{
            "id":        id,
            "encrypted": encrypted,
        })
        return nil
    })
    
    // Serve static files
    router.GET("/static/*filepath", func(ctx *chain.Context) error {
        filepath := ctx.GetParam("filepath")
        http.ServeFile(ctx.Writer, ctx.Request, "static"+filepath)
        return nil
    })
    
    // Error handler
    router.ErrorHandler = func(ctx *chain.Context, err error) {
        ctx.Json(map[string]string{"error": err.Error()})
        ctx.StatusInternalServerError()
    }
    
    http.ListenAndServe(":8080", router)
}
```

---

*End of API Reference*
