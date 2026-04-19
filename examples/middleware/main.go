// Package main demonstrates middleware patterns in Chain.
//
// Run with: go run main.go
// Then visiting:
//
//	curl -v http://localhost:8080/hello
//	curl -v -H "Authorization: Bearer secret" http://localhost:8080/api/data
//	curl -v http://localhost:8080/api/data          # should return 401
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/nidorx/chain"
)

func main() {
	router := chain.New()

	// ── 1. Global logging middleware (runs on every request) ───────────
	router.Use(func(ctx *chain.Context, next func() error) error {
		start := time.Now()

		err := next()

		duration := time.Since(start)
		log.Printf(
			"[%d] %s %-7s %s  (%v)",
			ctx.GetStatus(),
			ctx.Ip(),
			ctx.Method(),
			ctx.URL().Path,
			duration.Round(time.Microsecond),
		)

		return err
	})

	// ── 2. Request ID middleware ───────────────────────────────────────
	router.Use(func(ctx *chain.Context, next func() error) error {
		requestID := ctx.NewUID()
		ctx.Set("requestID", requestID)
		ctx.SetHeader("X-Request-ID", requestID)
		return next()
	})

	// ── 3. Simple handler ─────────────────────────────────────────────
	router.GET("/hello", func(ctx *chain.Context) error {
		ctx.Json(map[string]string{"message": "Hello, World!"})
		return nil
	})

	// ── 4. API routes with auth middleware ─────────────────────────────
	api := router.Group("/api")
	api.Use(authMiddleware)

	api.GET("/data", func(ctx *chain.Context) error {
		requestID, _ := ctx.Get("requestID")
		ctx.Json(map[string]string{
			"data":      "secret information",
			"requestID": requestID.(string),
		})
		return nil
	})

	// ── 5. Rate limiter demo ───────────────────────────────────────────
	limiter := NewRateLimiter(5, time.Second) // 5 requests per second

	router.GET("/limited", func(ctx *chain.Context) error {
		ip := ctx.Ip()
		if !limiter.Allow(ip) {
			ctx.Status(http.StatusTooManyRequests)
			ctx.Json(map[string]string{"error": "rate limit exceeded"})
			return nil
		}
		ctx.Json(map[string]string{"message": "request allowed"})
		return nil
	})

	// ── 6. CORS middleware demo ────────────────────────────────────────
	router.Use(func(ctx *chain.Context, next func() error) error {
		ctx.SetHeader("Access-Control-Allow-Origin", "*")
		ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if ctx.Method() == "OPTIONS" {
			ctx.NoContent()
			return nil
		}

		return next()
	})

	router.OPTIONS("/*", func(ctx *chain.Context) error {
		// handled by CORS middleware above
		return nil
	})

	// ── 7. Recovery middleware (catch panics) ─────────────────────────
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, rcv any) {
		log.Printf("[PANIC] recovered from: %v", rcv)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}

	router.GET("/panic", func(ctx *chain.Context) error {
		panic("this is a test panic")
	})

	log.Println("Middleware demo listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// authMiddleware checks for a valid Authorization header.
func authMiddleware(ctx *chain.Context, next func() error) error {
	token := ctx.GetHeader("Authorization")
	if token == "" {
		ctx.Status(http.StatusUnauthorized)
		ctx.Json(map[string]string{"error": "missing authorization"})
		return nil
	}

	// In a real application, validate JWT or session token here
	if token != "Bearer secret" {
		ctx.Status(http.StatusUnauthorized)
		ctx.Json(map[string]string{"error": "invalid token"})
		return nil
	}

	return next()
}

// RateLimiter is a simple per-key rate limiter.
type RateLimiter struct {
	counts map[string]int
	limit  int
	window time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		counts: make(map[string]int),
		limit:  limit,
		window: window,
	}
	// Periodically reset counts
	go func() {
		for {
			time.Sleep(window)
			rl.counts = make(map[string]int)
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	count := rl.counts[key]
	if count >= rl.limit {
		return false
	}
	rl.counts[key] = count + 1
	return true
}
