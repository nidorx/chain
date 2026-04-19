// Package main demonstrates route grouping in Chain.
//
// Run with: go run main.go
// Then visiting:
//
//	curl http://localhost:8080/api/v1/users                                                             # requires auth
//	curl http://localhost:8080/api/v1/users/42                                                          # requires auth
//	curl http://localhost:8080/public/info                                                              # no auth needed
//	curl -H "Authorization: Bearer secret" http://localhost:8080/api/v1/users/42                        # ok
//	curl -H "Authorization: Bearer secret" http://localhost:8080/api/admin/dashboard                    # requires X-Role
//	curl -H "X-Role: admin" -H "Authorization: Bearer secret" http://localhost:8080/api/admin/dashboard # ok
package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/nidorx/chain"
)

func main() {
	router := chain.New()

	// ── Public routes (no authentication) ──────────────────────────────

	public := router.Group("/public")
	{
		public.GET("/info", func(ctx *chain.Context) error {
			ctx.Json(map[string]string{
				"app":     "Chain Route Groups Demo",
				"version": "1.0.0",
			})
			return nil
		})

		public.GET("/status", func(ctx *chain.Context) error {
			ctx.Json(map[string]string{"status": "running"})
			return nil
		})
	}

	// ── API routes (authentication required) ───────────────────────────

	api := router.Group("/api")

	// Auth middleware applied to entire /api group
	api.Use(func(ctx *chain.Context, next func() error) error {
		token := ctx.GetHeader("Authorization")
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			ctx.Unauthorized(map[string]string{"error": "missing or invalid Authorization header"})
			return nil
		}
		// In a real app, validate the token here
		return next()
	})

	v1 := api.Group("/v1")
	{
		v1.GET("/users", func(ctx *chain.Context) error {
			ctx.Json(map[string]any{
				"users": []map[string]string{
					{"id": "1", "name": "Alice"},
					{"id": "2", "name": "Bob"},
				},
			})
			return nil
		})

		v1.POST("/users", func(ctx *chain.Context) error {
			ctx.Status(http.StatusCreated)
			ctx.Json(map[string]string{"message": "user created", "status": "created"})
			return nil
		})

		v1.GET("/users/:id", func(ctx *chain.Context) error {
			id := ctx.GetParam("id")
			ctx.Json(map[string]string{"id": id, "name": "User " + id})
			return nil
		})
	}

	v2 := api.Group("/v2")
	{
		v2.GET("/users", func(ctx *chain.Context) error {
			ctx.Json(map[string]any{
				"version": "v2",
				"data": []map[string]string{
					{"id": "1", "name": "Alice", "email": "alice@example.com"},
					{"id": "2", "name": "Bob", "email": "bob@example.com"},
				},
			})
			return nil
		})
	}

	// ── Admin routes (nested group with additional middleware) ─────────

	admin := api.Group("/admin")
	admin.Use(func(ctx *chain.Context, next func() error) error {
		// Additional admin-level check
		role := ctx.GetHeader("X-Role")
		if role != "admin" {
			ctx.Forbidden(map[string]string{"error": "admin role required"})
			return nil
		}
		return next()
	})

	admin.GET("/dashboard", func(ctx *chain.Context) error {
		ctx.Json(map[string]string{"message": "Welcome, admin!"})
		return nil
	})

	log.Println("Route groups server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
