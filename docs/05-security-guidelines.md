# Chain Framework - Security Guidelines

**Version:** 1.0.0 (Draft)  
**Last Updated:** April 18, 2026

---

## Table of Contents

1. [Introduction](#introduction)
2. [Known Vulnerabilities](#known-vulnerabilities)
3. [Security Best Practices](#security-best-practices)
4. [Cryptographic Guidelines](#cryptographic-guidelines)
5. [Input Validation](#input-validation)
6. [Authentication & Authorization](#authentication--authorization)
7. [Secure Configuration](#secure-configuration)
8. [Incident Response](#incident-response)
9. [Security Checklist](#security-checklist)

---

## Introduction

This document outlines security considerations for applications built with the Chain framework. It covers known vulnerabilities, recommended mitigations, and security best practices.

### Threat Model

Chain applications face the following primary threats:

1. **Denial of Service (DoS)** - Resource exhaustion through goroutine leaks, memory leaks
2. **Injection Attacks** - Malicious input data
3. **Authentication Bypass** - Token theft, weak validation
4. **Data Exposure** - Unencrypted data, weak cryptography
5. **Route Confusion** - Path traversal, parameter injection

---

## Known Vulnerabilities

### CRITICAL - Goroutine Leak (CVE Pending)

**Severity:** HIGH  
**Status:** Unpatched  
**Component:** `router.go:337-341`

**Description:**
A goroutine is spawned for each request to clean up the context when the connection closes. Under certain conditions (client disconnects, network errors), this goroutine may never terminate, leading to resource exhaustion.

**Vulnerable Code:**
```go
go func() {
    <-ctx.Request.Context().Done()
    r.poolPutContext(ctx)
}()
```

**Impact:**
- Memory exhaustion under high load
- Goroutine accumulation (thousands per second)
- Application crash (out of memory)

**Mitigation (Workaround):**
```go
// In your application, wrap the router
type SafeRouter struct {
    *chain.Router
}

func (s *SafeRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Set a timeout on the request context
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    r = r.WithContext(ctx)
    s.Router.ServeHTTP(w, r)
}
```

**Fix Status:** Planned for Phase 1 of evolution roadmap

---

### HIGH - GetHeader() Bug (CVE Pending)

**Severity:** HIGH  
**Status:** Unpatched  
**Component:** `context_request.go:108-111`

**Description:**
The `GetHeader()` function reads from response headers instead of request headers, breaking all header-based functionality.

**Impact:**
- Authentication header extraction fails
- CORS header reading fails
- Content-Type detection broken
- All header-based security mechanisms affected

**Fix:**
```go
// In your application, use directly:
contentType := ctx.Request.Header.Get("Content-Type")

// Or patch the function:
func GetHeaderFixed(ctx *chain.Context, key string) string {
    return ctx.Request.Header.Get(key)
}
```

**Fix Status:** Planned for Phase 1 of evolution roadmap

---

### MEDIUM - Weak PBKDF2 Defaults

**Severity:** MEDIUM  
**Status:** Documented  
**Component:** `crypto/key_generator.go`

**Description:**
Default PBKDF2 iterations count is 1000, which is insufficient for modern security requirements.

**Recommendation:**
```go
// Always specify iteration count explicitly
key := chain.Crypto().KeyGenerate(
    secret,
    salt,
    216000,  // Minimum recommended
    32,
    "sha256",
)
```

---

## Security Best Practices

### 1. Secret Key Management

**DO:**
```go
// Use strong, random keys (32 bytes minimum)
import "crypto/rand"

func generateSecretKey() string {
    key := make([]byte, 32)
    rand.Read(key)
    return base64.StdEncoding.EncodeToString(key)
}

// Set at application startup
chain.SetSecretKeyBase(generateSecretKey())
```

**DON'T:**
```go
// ❌ Hardcoded keys
chain.SetSecretKeyBase("my-secret-key")

// ❌ Weak keys
chain.SetSecretKeyBase("123456")

//  keys from environment without validation
chain.SetSecretKeyBase(os.Getenv("SECRET"))
```

**Best Practice:**
```go
func initSecretKey() error {
    keyBase := os.Getenv("SECRET_KEY_BASE")
    
    if keyBase == "" {
        // Generate random key for development
        key := make([]byte, 32)
        if _, err := rand.Read(key); err != nil {
            return err
        }
        keyBase = base64.StdEncoding.EncodeToString(key)
        log.Println("[WARNING] Generated random secret key. Set SECRET_KEY_BASE for production")
    }
    
    // Validate key
    key := []byte(keyBase)
    if len(key) < 16 {
        return fmt.Errorf("secret key too short: %d bytes (minimum 16)", len(key))
    }
    
    return chain.SetSecretKeyBase(keyBase)
}
```

---

### 2. Request Size Limits

**Problem:** Chain does not limit request body size by default.

**Solution:**
```go
// Add middleware to limit request size
router.Use(func(ctx *chain.Context, next func() error) error {
    // Limit to 10MB
    ctx.Request.Body = http.MaxBytesReader(
        ctx.Writer,
        ctx.Request.Body,
        10<<20, // 10MB
    )
    return next()
})
```

---

### 3. Rate Limiting

Chain does not include built-in rate limiting. Implement as middleware:

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.Mutex
    r        rate.Limit
    b        int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        r:        r,
        b:        b,
    }
}

func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    if limiter, exists := rl.limiters[key]; exists {
        return limiter
    }
    
    limiter := rate.NewLimiter(rl.r, rl.b)
    rl.limiters[key] = limiter
    return limiter
}

// Usage
limiter := NewRateLimiter(10, 20) // 10 requests/second, burst 20

router.Use(func(ctx *chain.Context, next func() error) error {
    ip := ctx.Ip()
    if !limiter.GetLimiter(ip).Allow() {
        ctx.TooManyRequests()
        return nil
    }
    return next()
})
```

---

### 4. CORS Configuration

Implement secure CORS:

```go
router.Use(func(ctx *chain.Context, next func() error) error {
    origin := ctx.GetHeader("Origin")
    
    // Whitelist allowed origins
    allowedOrigins := map[string]bool{
        "https://example.com": true,
    }
    
    if !allowedOrigins[origin] {
        // Don't set CORS header for untrusted origins
        return next()
    }
    
    ctx.SetHeader("Access-Control-Allow-Origin", origin)
    ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
    ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
    ctx.SetHeader("Access-Control-Allow-Credentials", "true")
    ctx.SetHeader("Access-Control-Max-Age", "86400") // 24 hours
    
    if ctx.Method() == "OPTIONS" {
        ctx.OK()
        return nil
    }
    
    return next()
})
```

---

### 5. Security Headers

Add security headers middleware:

```go
router.Use(func(ctx *chain.Context, next func() error) error {
    // Prevent MIME type sniffing
    ctx.SetHeader("X-Content-Type-Options", "nosniff")
    
    // Prevent clickjacking
    ctx.SetHeader("X-Frame-Options", "DENY")
    
    // Enable XSS protection
    ctx.SetHeader("X-XSS-Protection", "1; mode=block")
    
    // Strict Transport Security
    ctx.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    
    // Content Security Policy (customize for your application)
    ctx.SetHeader("Content-Security-Policy", "default-src 'self'")
    
    // Referrer Policy
    ctx.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
    
    return next()
})
```

---

### 6. Error Handling Security

**Problem:** Internal error details may be exposed to clients.

**Solution:**
```go
// Global error handler that sanitizes errors
router.ErrorHandler = func(ctx *chain.Context, err error) {
    // Log full error for debugging
    log.Printf("[ERROR] %v", err)
    
    // Check if it's a validation error
    if validationErr, ok := err.(chain.SliceValidationError); ok {
        // Return validation errors to client
        ctx.Json(map[string]any{
            "error": "Validation failed",
            "details": validationErr.Error(),
        })
        ctx.StatusBadRequest()
        return
    }
    
    // Return generic error for other errors
    ctx.Json(map[string]string{
        "error": "Internal Server Error",
    })
    ctx.StatusInternalServerError()
}
```

---

## Cryptographic Guidelines

### Algorithm Selection

| Purpose | Algorithm | Key Size | Notes |
|---------|-----------|----------|-------|
| Symmetric encryption | AES-GCM | 128/256 bits | Authenticated encryption |
| Key derivation | PBKDF2 | - | Use 216,000+ iterations |
| Message signing | HMAC-SHA256 | 256 bits | Constant-time verification |
| Hashing (non-security) | XXH64 | - | Fast, not cryptographically secure |
| Hashing (security) | SHA-256 | - | Use for checksums |
| **DO NOT USE** | MD5 | - | Cryptographically broken |

### Key Rotation

Use Keyring for key rotation:

```go
// Create keyring
keyring := chain.NewKeyring("salt", 216000, 32, "sha256")

// Encrypt with current primary key
encrypted, _ := keyring.Encrypt(data, nil)

// Decrypt (tries all keys in rotation)
decrypted, _ := keyring.Decrypt(encrypted, nil)

// Rotate keys
func rotateKeys() {
    // Add new key (becomes primary)
    newKey := generateKey()
    keyring.AddKey(newKey)
    
    // Old keys remain available for decryption
    // Eventually remove old keys
}
```

### Secure Message Exchange

**Sender:**
```go
// Encrypt and sign message
message := []byte("Sensitive data")
aad := []byte("context") // Additional authenticated data

encrypted, _ := chain.Crypto().MessageEncrypt(
    []byte(chain.SecretKeyBase()),
    message,
    aad,
)

// Send encrypted message
```

**Receiver:**
```go
// Decrypt and verify message
decrypted, err := chain.Crypto().MessageDecrypt(
    []byte(chain.SecretKeyBase()),
    []byte(encryptedMessage),
    []byte("context"),
)

if err != nil {
    // Tampered or invalid
    log.Printf("Message verification failed: %v", err)
    return
}

// Use decrypted message
```

---

## Input Validation

### Route Parameter Validation

```go
router.GET("/users/:id", func(ctx *chain.Context) error {
    id := ctx.GetParam("id")
    
    // Validate format
    if !isValidID(id) {
        ctx.BadRequest()
        return nil
    }
    
    // Validate length
    if len(id) > 36 {
        ctx.BadRequest()
        return nil
    }
    
    // ... handler logic
})
```

### Request Body Validation

Use struct tags with validator:

```go
type CreateUserRequest struct {
    Name     string `json:"name" binding:"required,min=3,max=100"`
    Email    string `json:"email" binding:"required,email,max=255"`
    Password string `json:"password" binding:"required,min=8,max=128"`
    Age      int    `json:"age" binding:"min=0,max=150"`
    Website  string `json:"website" binding:"omitempty,url"`
}

router.POST("/users", func(ctx *chain.Context) error {
    var req CreateUserRequest
    
    if err := ctx.BindJSON(&req); err != nil {
        return err // Returns 400 with validation errors
    }
    
    // req is validated, proceed with handler logic
})
```

### Query Parameter Validation

```go
type ListUsersRequest struct {
    Page     int    `query:"page" binding:"min=1"`
    Limit    int    `query:"limit" binding:"min=1,max=100"`
    Sort     string `query:"sort" binding:"omitempty,oneof=name email created_at"`
    Order    string `query:"order" binding:"omitempty,oneof=asc desc"`
}

router.GET("/users", func(ctx *chain.Context) error {
    var req ListUsersRequest
    
    if err := ctx.BindQuery(&req); err != nil {
        return err
    }
    
    // Use validated parameters
})
```

---

## Authentication & Authorization

### Token-Based Authentication

```go
type AuthMiddleware struct {
    secret string
}

func (m *AuthMiddleware) Handle(ctx *chain.Context, next func() error) error {
    authHeader := ctx.Request.Header.Get("Authorization")
    if authHeader == "" {
        ctx.Json(map[string]string{"error": "Missing authorization header"})
        ctx.Status(http.StatusUnauthorized)
        return nil
    }
    
    if !strings.HasPrefix(authHeader, "Bearer ") {
        ctx.Json(map[string]string{"error": "Invalid authorization format"})
        ctx.Status(http.StatusUnauthorized)
        return nil
    }
    
    token := strings.TrimPrefix(authHeader, "Bearer ")
    
    // Verify token using Chain's crypto
    decoded, err := chain.Crypto().MessageVerify(
        []byte(m.secret),
        []byte(token),
    )
    
    if err != nil {
        ctx.Json(map[string]string{"error": "Invalid token"})
        ctx.Status(http.StatusUnauthorized)
        return nil
    }
    
    // Store user info in context
    ctx.Set("user", decoded)
    
    return next()
}

// Usage
auth := &AuthMiddleware{secret: chain.SecretKeyBase()}
api := router.Group("/api")
api.Use(auth)
```

### Signed Tokens

**Token Generation:**
```go
router.POST("/login", func(ctx *chain.Context) error {
    var req LoginRequest
    ctx.BindJSON(&req)
    
    if !authenticateUser(req.Username, req.Password) {
        ctx.Unauthorized()
        return nil
    }
    
    // Create token payload
    payload := map[string]any{
        "user_id":   "123",
        "username":  req.Username,
        "expires":   time.Now().Add(24 * time.Hour).Unix(),
    }
    
    // Serialize payload
    payloadJSON, _ := json.Marshal(payload)
    
    // Sign token
    token := chain.Crypto().MessageSign(
        []byte(chain.SecretKeyBase()),
        payloadJSON,
        "sha256",
    )
    
    ctx.Json(map[string]string{"token": token})
    return nil
})
```

**Token Verification:**
```go
func verifyToken(ctx *chain.Context) (map[string]any, error) {
    token := ctx.Request.Header.Get("Authorization")
    
    decoded, err := chain.Crypto().MessageVerify(
        []byte(chain.SecretKeyBase()),
        []byte(token),
    )
    
    if err != nil {
        return nil, err
    }
    
    var payload map[string]any
    json.Unmarshal(decoded, &payload)
    
    // Check expiration
    if expires, ok := payload["expires"].(float64); ok {
        if time.Now().Unix() > int64(expires) {
            return nil, errors.New("token expired")
        }
    }
    
    return payload, nil
}
```

---

## Secure Configuration

### Production Checklist

```go
func configureRouter() *chain.Router {
    router := chain.New()
    
    // Set secret key (REQUIRED)
    if err := initSecretKey(); err != nil {
        log.Fatalf("Failed to initialize secret key: %v", err)
    }
    
    // Configure panic handler
    router.PanicHandler = func(w http.ResponseWriter, r *http.Request, rcv any) {
        log.Printf("[PANIC] %v", rcv)
        // Don't expose internal details
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    }
    
    // Configure error handler
    router.ErrorHandler = func(ctx *chain.Context, err error) {
        log.Printf("[ERROR] %v", err)
        ctx.Json(map[string]string{"error": "Internal Server Error"})
        ctx.StatusInternalServerError()
    }
    
    // Configure 404 handler
    router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusNotFound)
        w.Write([]byte(`{"error":"Not Found"}`))
    })
    
    // Add security middleware
    router.Use(securityHeadersMiddleware)
    router.Use(rateLimiterMiddleware)
    router.Use(requestSizeLimitMiddleware)
    
    return router
}
```

### HTTPS Configuration

Always use HTTPS in production:

```go
func main() {
    router := configureRouter()
    
    // Redirect HTTP to HTTPS
    go http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
    }))
    
    // Serve HTTPS
    log.Fatal(http.ListenAndServeTLS(":443", "cert.pem", "key.pem", router))
}
```

---

## Incident Response

### Security Issue Detected

1. **Contain** - Disable affected endpoints/features
2. **Assess** - Determine scope and impact
3. **Remediate** - Apply fixes or workarounds
4. **Verify** - Test that fix resolves the issue
5. **Monitor** - Watch for recurrence
6. **Document** - Record incident and lessons learned

### Reporting Security Issues

If you discover a security vulnerability in Chain:

1. **DO NOT** open a public issue
2. Email: [security contact - to be configured]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

---

## Security Checklist

### Before Deployment

- [ ] Secret key set and stored securely
- [ ] PBKDF2 iterations >= 216,000
- [ ] MD5 not used for security purposes
- [ ] Request size limits configured
- [ ] Rate limiting enabled
- [ ] CORS properly configured
- [ ] Security headers added
- [ ] Error messages sanitized
- [ ] Panic handler configured
- [ ] HTTPS enabled
- [ ] Dependencies updated
- [ ] Input validation implemented
- [ ] Authentication/authorization tested
- [ ] Logging enabled for security events
- [ ] Backup and recovery tested

### Periodic Review

- [ ] Rotate secret keys
- [ ] Review access controls
- [ ] Audit logs for anomalies
- [ ] Update dependencies
- [ ] Review rate limits
- [ ] Test authentication bypass scenarios
- [ ] Verify TLS configuration
- [ ] Check for new vulnerabilities in dependencies

---

*End of Security Guidelines*
