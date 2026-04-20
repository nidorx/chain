# Chain Framework - Architecture Overview

- **Version:** 1.0.0
- **Last Updated:** April 19, 2026


## Table of Contents

1. [System Overview](#system-overview)
2. [System Architecture Diagram](#system-architecture-diagram)
3. [Component Relationships](#component-relationships)
4. [Request Lifecycle](#request-lifecycle)
5. [Routing Algorithm](#routing-algorithm)
6. [Context Management](#context-management)
7. [Middleware Execution](#middleware-execution)
8. [Design Decisions](#design-decisions)
9. [Trade-offs](#trade-offs)
10. [Performance Characteristics](#performance-characteristics)


## System Overview

Chain is a high-performance HTTP router and distributed systems toolkit for Go. It is designed around three core subsystems:

1. **HTTP Router** — Segment-based routing with static route caching and priority matching
2. **Cryptographic Utilities** — AES-GCM encryption, PBKDF2 key derivation, HMAC signing, and key rotation
3. **Real-time Communication** — Cluster-aware pub/sub and multiplexed socket channels over SSE

### Package Layout

```
github.com/nidorx/chain/
├── chain.go                     # Router factory (chain.New())
├── router.go                    # Core router implementation
├── router_group.go              # Route grouping
├── registry.go                  # Route registration and lookup
├── route.go                     # Route and middleware types
├── route_info.go                # Route parsing and matching
├── route_storage.go             # Segment-based route storage
├── context.go                   # Request/response context
├── context_request.go           # Request access methods
├── context_response.go          # Response writing methods
├── context_binding.go           # Data binding orchestration
├── context_binding_*.go         # Specific binding implementations
├── context_binding_validator.go # Validation integration
├── response_writer.go           # ResponseWriter wrapper
├── middleware_helpers.go        # Middleware utilities
├── crypto.go                    # Crypto shortcuts
├── crypto_keyring.go            # Keyring factory
├── errors.go                    # Error types and validation
├── utils.go                     # Hashing, serialization, UID generation
├── node.go                      # Node identity management
├── crypto/                      # Cryptographic primitives
│   ├── crypto.go                # Crypto interface
│   ├── aes-gcm.go               # AES-GCM encryption
│   ├── key_generator.go         # PBKDF2 key derivation
│   ├── keyring.go               # Key rotation
│   ├── message_encryptor.go     # Authenticated encryption
│   ├── message_verifier.go      # HMAC message signing
│   └── utils.go                 # Crypto utilities
├── pubsub/                      # Publish/Subscribe system
│   ├── pubsub.go                # Core pub/sub
│   ├── adapter.go               # Adapter interface
│   ├── message.go               # Message types
│   ├── options.go               # Broadcast options
│   ├── crypto.go                # Message encryption
│   └── compression.go           # Message compression
├── socket/                      # Real-time channels
│   ├── handler.go               # Socket handler
│   ├── channel.go               # Channel management
│   ├── session.go               # Session management
│   ├── socket.go                # Socket implementation
│   ├── message.go               # Message types
│   ├── transport.go             # Transport interface
│   ├── transport_sse.go         # SSE transport
│   └── message_serializer.go    # Message serialization
├── pkg/                         # Shared utilities
│   ├── pathclean.go             # URL path cleaning
│   └── wildcard_store.go        # Wildcard pattern matching
└── middlewares/session/         # Session middleware
    ├── manager.go               # Session manager
    ├── session.go               # Session data
    ├── store.go                 # Store interface
    └── store_cookie.go          # Cookie-based store
```

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        net/http Server                          │
│                   (http.ListenAndServe)                         │
└────────────────────────┬────────────────────────────────────────┘
                         │ http.Handler (ServeHTTP)
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Router                                 │
│  ┌──────────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ Registries       │  │ Context Pool │  │ Configuration     │  │
│  │ (per HTTP method)│  │ (sync.Pool)  │  │ (redirects,       │  │
│  │                  │  │              │  │  OPTIONS, CORS)   │  │
│  └────────┬─────────┘  └──────┬───────┘  └───────────────────┘  │
│           │                   │                                 │
│           ▼                   ▼                                 │
│  ┌──────────────────┐  ┌──────────────┐                         │
│  │ RouteStorage     │  │ Handlers     │                         │
│  │ (segment-based)  │  │ (Panic,      │                         │
│  │ + Static map     │  │  Error,      │                         │
│  │                  │  │  NotFound)   │                         │
│  └──────────────────┘  └──────────────┘                         │
└────────────────────────┬────────────────────────────────────────┘
                         │ route.Dispatch(ctx)
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Middleware Chain                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌──────────────┐  │
│  │ MW 1    │───>│ MW 2    │───>│ MW N    │───>│ Route Handler│  │
│  │ (Log)   │    │ (Auth)  │    │ (CORS)  │    │ (Business)   │  │
│  └─────────┘    └─────────┘    └─────────┘    └──────────────┘  │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Context                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │ Request     │  │ Response     │  │ Route Info             │  │
│  │ (params,    │  │ Writer       │  │ (matched route,        │  │
│  │  query,     │  │ (status,     │  │  params, handler)      │  │
│  │  body)      │  │  headers)    │  │                        │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │ Data Store  │  │ Crypto       │  │ Hooks                  │  │
│  │ (key-value) │  │ (encrypt,    │  │ (BeforeSend,           │  │
│  │             │  │  sign, keys) │  │  AfterSend)            │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└────────────────────────┬────────────────────────────────────────┘
                         │
              ┌──────────┴──────────┐
              ▼                     ▼
┌─────────────────────┐ ┌─────────────────────────────────────────┐
│  Data Binding       │ │  Subsystems                             │
│  ┌───────────────┐  │ │  ┌──────────┐ ┌────────┐ ┌───────────┐  │
│  │ JSON/XML      │  │ │  │ Pub/Sub  │ │ Socket │ │ Session   │  │
│  │ Form/Query    │  │ │  │ (cluster)│ │(SSE)   │ │ (cookie)  │  │
│  │ Path/Header   │  │ │  └──────────┘ └────────┘ └───────────┘  │
│  └───────────────┘  │ │  ┌──────────────────────────────────┐   │
│  ┌───────────────┐  │ │  │ Crypto (AES-GCM, PBKDF2, HMAC)   │   │
│  │ Validator     │  │ │  └──────────────────────────────────┘   │
│  │ (go-playgnd)  │  │ │                                         │
│  └───────────────┘  │ │                                         │
└─────────────────────┘ └─────────────────────────────────────────┘
```


## Component Relationships

### Router → Registry → RouteStorage

The Router maintains one **Registry** per HTTP method (GET, POST, etc.). Each Registry contains:

- **Static map** — O(1) lookup for routes without parameters
- **RouteStorage** — Segment-based storage for dynamic routes (parameters, wildcards)
- **Middleware list** — Middlewares matched against route patterns

```
Router
  ├── Registry["GET"]
  │     ├── Static{ "/health": Route }
  │     ├── Static{ "/ping": Route }
  │     ├── RouteStorage{ segment_count → [Route...] }
  │     └── Middlewares[ {path: "/api/*", handler: authMW} ]
  ├── Registry["POST"]
  │     ├── ...
  └── Registry["DELETE"]
        └── ...
```

### Router → Context Pool

The Router uses `sync.Pool` to recycle Context objects, reducing GC pressure under load:

```
Router.contextPool (sync.Pool)
  ├── Get() → reuse existing Context or allocate new
  └── Put() → reset fields, return to pool
```

### Context → Crypto

Each Context holds a reference to the crypto implementation, allowing per-request encryption operations:

```
Context
  ├── Crypto → chain.Crypto() (shared global instance)
  └── Uses SecretKeyBase for key derivation
```

### Socket → Channel → PubSub

Socket channels integrate with the pub/sub system for cluster-wide message distribution:

```
Handler
  ├── Transport (SSE)
  │     └── Session
  │           └── Socket (per topic)
  │                 └── Channel
  │                       ├── Join/Leave handlers
  │                       ├── HandleIn (client → server)
  │                       ├── HandleOut (server → client)
  │                       └── Broadcast → PubSub.Broadcast
  └── Channels[] (registered channel factories)
```


## Request Lifecycle

### Phase 1: Reception

```
HTTP Request arrives
  │
  ▼
Router.ServeHTTP(w, r)
  │
  ├─ Creates ResponseWriterSpy (tracks write state)
  ├─ Gets Context from pool (or allocates new)
  ├─ Initializes Context with request data
  └─ Parses URL path into segments
```

### Phase 2: Route Matching

```
Registry.findHandle(ctx)
  │
  ├─ Check static routes (O(1) map lookup)
  │    └─ If match → return Route
  │
  └─ RouteStorage.lookup(ctx)
       │
       ├─ Iterate from path segment count down to 1
       │    └─ For each route group:
       │         └─ FastMatch() — compare static segments
       │              └─ If match → extract params → return Route
       │
       └─ No match → return nil
```

### Phase 3: Middleware Chain

```
Route.Dispatch(ctx)
  │
  ├─ Collect matching middlewares (by path pattern)
  ├─ Build nested handler chain:
  │     MW1 → MW2 → ... → MWN → Handler
  │
  └─ Execute chain:
       MW1.before → MW2.before → ... → Handler → ... → MW2.after → MW1.after
```

### Phase 4: Handler Execution

```
Route Handler(ctx)
  │
  ├─ Access request data (params, body, headers)
  ├─ Data binding (BindJSON, BindForm, etc.)
  ├─ Validation (automatic with binding tags)
  ├─ Business logic
  └─ Write response (Json, Write, Status, etc.)
```

### Phase 5: Response & Cleanup

```
Response written
  │
  ├─ BeforeSend hooks execute
  ├─ Response sent to client
  ├─ AfterSend hooks execute
  └─ Context returned to pool
```


## Routing Algorithm

Chain uses a **segment-based routing algorithm** with static route caching. This differs from traditional radix tree approaches (like httprouter).

### Route Registration

1. Path is cleaned and split into segments: `/api/v1/users/:id` → `["api", "v1", "users", ":id"]`
2. Route priority is calculated based on segment types:
   - Static segment weight = 3
   - Parameter segment (`:name`) weight = 2
   - Wildcard segment (`*`) weight = 1
   - Formula: `priority = Σ(segment_weight × height²)`
3. Route is stored in:
   - **Static map** (if no parameters) for O(1) lookup
   - **RouteStorage** (if dynamic), indexed by segment count

### Route Lookup

1. Check static map first (fastest path)
2. If not found, search RouteStorage:
   - Start at routes matching the path's segment count
   - Work downward to fewer segments (wildcards can match more)
   - FastMatch skips parameter/wildcard segments, comparing only static ones
   - First match wins (priority order within each segment group)

### Example

```
Routes:
  GET /api/v1/users           → static (cached)
  GET /api/:version/users     → dynamic (4 segments)
  GET /api/:version/users/:id → dynamic (5 segments)
  GET /files/*filepath        → dynamic (2 segments, wildcard)

Lookup "GET /api/v1/users/123":
  1. Not in static map (has params)
  2. RouteStorage: check 5-segment routes
  3. Match /api/:version/users/:id → extract version="v1", id="123"
  4. Return matched route
```


## Context Management

### Pool-Based Recycling

```
Request arrives
  │
  ├─ pool.Get()
  │    ├─ Reuse existing Context (if available)
  │    └─ Allocate new (if pool empty)
  │
  ├─ Initialize Context fields
  │
  ├─ Process request...
  │
  └─ pool.Put()
       └─ Reset fields, return to pool
```

### Context Hierarchy

Contexts support parent-child relationships for sub-requests and internal redirects:

```
Root Context (HTTP request)
  │
  ├─ Child Context 1 (middleware with custom params)
  │     └─ Grandchild Context 1.1
  │
  └─ Child Context 2 (internal redirect)
```

Children inherit parent's route info and router reference but have independent data stores.


## Middleware Execution

### Registration

```go
// Global (all methods, all paths)
router.Use(middleware)

// Path-scoped (all methods, specific path)
router.Use("/api/*", middleware)

// Method + path scoped
router.Use("GET", "/api/*", middleware)

// Group-scoped
api.Use(middleware) // applies to all routes in group
```

### Matching Algorithm

Middlewares are matched against routes using pattern matching:

1. Exact path match → always applies
2. Wildcard middleware (`/api/*`) → matches all sub-paths (`/api/v1/users`)
3. Method-restricted middleware → only applies to specified HTTP methods

### Execution Order

```
Request → MW₁ → MW₂ → ... → MWₙ → Handler → MWₙ → ... → MW₂ → MW₁ → Response
          (before)                           (after)
```

If any middleware returns without calling `next()`, the chain short-circuits and the handler is never executed.

### Shipped Middlewares

Chain includes several production-ready middlewares:

| Middleware | Package | Description |
|-----------|---------|-------------|
| CORS | `middlewares/cors` | Full-featured CORS with wildcard, regex, and context-aware origin validation |
| Logger | `middlewares/logger` | Structured request logging with `log/slog` |
| Recovery | `middlewares/recovery` | Panic recovery with stack traces |
| Limiter | `middlewares/limiter` | Request body size limiting |
| Session | `middlewares/session` | Cookie-based encrypted sessions |
| Timeout | `middlewares/timeout` | Request timeout enforcement with context cancellation |


### Graceful Shutdown

The `chain.Server` type provides production-ready server lifecycle management:

```
┌─────────────────────────────────────────────────────┐
│                    chain.Server                      │
│  ┌──────────────┐  ┌──────────────────────────────┐  │
│  │ http.Server  │  │ ShutdownConfig               │  │
│  │ (Addr,       │  │ - Timeout (default 30s)      │  │
│  │  Handler)    │  │ - Signals (SIGINT, SIGTERM)  │  │
│  └──────┬───────┘  └──────────────────────────────┘  │
│         │                                            │
│  ┌──────┴───────┐  ┌──────────────────────────────┐  │
│  │ Lifecycle    │  │ State                        │  │
│  │ Hooks        │  │ - shutting (atomic)          │  │
│  │ - OnShutdown │  │ - stopChan                   │  │
│  │ - OnStop     │  └──────────────────────────────┘  │
│  └──────────────┘                                    │
└─────────────────────────────────────────────────────┘
```

The shutdown sequence:

1. **Signal reception** — `waitForShutdownSignal()` blocks on `os.Signal` channel (default: `SIGINT`, `SIGTERM`)
2. **`OnShutdown` hook** — Invoked immediately, before any waiting
3. **`http.Server.Shutdown(ctx)`** — Stops accepting new connections, waits for in-flight requests with context timeout
4. **`OnStop` hook** — Invoked after all requests complete or timeout
5. **`stopChan` closed** — Unblocks any goroutines waiting on `server.Wait()`

The `GracefulMiddleware` integrates with this lifecycle by checking `server.IsShuttingDown()` on each request and setting `Connection: close` to drain keep-alive connections.


## Design Decisions

### 1. Segment-Based Routing vs Radix Tree

**Decision:** Use segment-based routing instead of radix tree.

**Rationale:**
- Simpler to implement and maintain
- Flexible wildcard and parameter matching
- Static route caching covers the common case (O(1))
- Wildcard propagation is straightforward

**Trade-offs:**
- ✅ Simpler code, easier to extend
- ✅ More flexible pattern matching
- ❌ Slower than radix tree for very large route sets (1000+ routes)
- ❌ Higher memory usage for route storage

### 2. Context Pooling

**Decision:** Use `sync.Pool` for Context recycling.

**Rationale:**
- Reduces garbage collection pressure
- Standard Go pattern for request-scoped objects
- Significant performance improvement under load

**Trade-offs:**
- ✅ Better throughput and latency
- ❌ Must carefully reset all fields on return
- ❌ Risk of data leaks if fields aren't cleared

### 3. Multiple Handler Signatures

**Decision:** Support `func(*Context)`, `func(*Context) error`, `http.Handler`, `http.HandlerFunc`, `func(http.ResponseWriter, *http.Request)`, etc.

**Rationale:**
- Easy migration from `net/http`
- Compatibility with existing middleware
- Flexibility for different use cases

**Trade-offs:**
- ✅ Developer-friendly, low friction
- ❌ Runtime type assertion overhead
- ❌ More complex implementation

### 4. Global Crypto State

**Decision:** Store secret keys globally via `SetSecretKeyBase()`.

**Rationale:**
- Convenient single-point configuration
- Keyring syncs automatically with SecretKeyBase changes
- Matches Rails' ActiveSupport::MessageEncryptor pattern

**Trade-offs:**
- ✅ Simple API, easy to use
- ❌ Not ideal for multi-tenant applications
- ❌ Harder to test in isolation (requires key setup/teardown)

### 5. Automatic Validation on Bind

**Decision:** `Bind*` methods automatically validate and return 400 on failure; `ShouldBind*` methods return errors without setting status.

**Rationale:**
- Common case (API server) is one-liner
- Advanced case (custom error formatting) is still possible

**Trade-offs:**
- ✅ Fast path for common usage
- ❌ Two sets of methods to learn


## Trade-offs

### Performance vs Flexibility

Chain prioritizes flexibility over raw performance in some areas:

| Area | Trade-off |
|------|-----------|
| Handler signatures | Runtime type checking vs compile-time safety |
| Multiple middleware formats | Flexibility vs performance |
| Global crypto state | Convenience vs testability |
| Segment-based routing | Simplicity vs radix tree speed |

### Memory vs Speed

| Component | Strategy |
|-----------|----------|
| Static routes | Cached in map (more memory, O(1) lookup) |
| Context objects | Pooled (more memory, fewer allocations) |
| Route params | Fixed-size arrays (32 slots, stack-allocated) |
| Path segments | Pre-parsed integers into path string |

### Simplicity vs Features

| Feature | Approach |
|---------|----------|
| Routing | Segment-based (simpler than radix tree) |
| Binding | Automatic with fallback (ShouldBind*) |
| Validation | Integrated via go-playground/validator |
| Crypto | High-level API over stdlib |
| Pub/Sub | In-memory with adapter interface |


## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| Static route lookup | O(1) | Map lookup |
| Parameter route lookup | O(n × s) | n = routes at segment level, s = segments |
| Wildcard route lookup | O(n × s) | Same as parameter |
| Middleware matching | O(m) | m = registered middlewares |
| Context creation | O(1) | Pool get + reset |
| Parameter extraction | O(p) | p = number of parameters |
| JSON binding | O(body size) | JSON decode |

### Space Complexity

| Component | Complexity | Notes |
|-----------|-----------|-------|
| Static routes | O(s) | s = static route count |
| Dynamic routes | O(d) | d = dynamic route count |
| Middleware storage | O(m) | m = middleware count |
| Context pool | O(c) | c = concurrent requests |
| Per-request memory | O(1) | Fixed-size arrays |

### Optimization Opportunities

1. **Radix tree** for dynamic route lookup (future enhancement — Phase 4.3.1)
2. **String interning** for path segments
3. **Zero-allocation parameter extraction** via direct slice access
4. **Connection pooling** optimization for context recycling


*End of Architecture Overview*

For more details, see:
- [API Reference](03-api-reference.md)
- [Security Guidelines](05-security-guidelines.md)
- [Evolution Roadmap](02-evolution-roadmap.md)
