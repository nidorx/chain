// Package main demonstrates data binding in Chain.
//
// Run with: go run main.go
// Then try:
//
//	# JSON binding
//	curl -X POST http://localhost:8080/users \
//	  -H "Content-Type: application/json" \
//	  -d '{"name":"Alice","email":"alice@example.com"}'
//
//	# Query binding
//	curl "http://localhost:8080/search?q=chain&limit=10&page=1"
//
//	# Path binding
//	curl http://localhost:8080/users/42
//
//	# Header binding
//	curl -H "X-API-Key: my-secret-key" http://localhost:8080/headers
//
//	# Form binding
//	curl -X POST http://localhost:8080/login \
//	  -d "username=alice&password=secret123"
package main

import (
	"log"
	"net/http"

	"github.com/nidorx/chain"
)

// ── Structs for binding ────────────────────────────────────────────────

type CreateUser struct {
	Name  string `json:"name"  binding:"required,min=2"`
	Email string `json:"email" binding:"required,email"`
}

type SearchQuery struct {
	Query string `query:"q"     binding:"required"`
	Limit int    `query:"limit" binding:"min=1,max=100"`
	Page  int    `query:"page"  binding:"min=1"`
}

type LoginForm struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required,min=6"`
}

type APIKey struct {
	Key string `header:"X-API-Key" binding:"required"`
}

// ── Main ───────────────────────────────────────────────────────────────

func main() {
	router := chain.New()

	// 1. JSON binding (MustBindWith — returns 400 on error)
	router.POST("/users", func(ctx *chain.Context) error {
		var u CreateUser
		if err := ctx.BindJSON(&u); err != nil {
			return err // 400 with validation details
		}
		ctx.Status(http.StatusCreated)
		ctx.Json(map[string]any{
			"id":    ctx.NewUID(),
			"name":  u.Name,
			"email": u.Email,
		})
		return nil
	})

	// 2. JSON binding (ShouldBindWith — you handle the error)
	router.POST("/users/should", func(ctx *chain.Context) error {
		var u CreateUser
		if err := ctx.ShouldBindJSON(&u); err != nil {
			ctx.Json(map[string]string{"error": err.Error()})
			ctx.Status(http.StatusBadRequest)
			return nil
		}
		ctx.Json(u)
		return nil
	})

	// 3. Query binding
	router.GET("/search", func(ctx *chain.Context) error {
		var q SearchQuery
		if err := ctx.BindQuery(&q); err != nil {
			return err
		}
		ctx.Json(map[string]any{
			"query":   q.Query,
			"limit":   q.Limit,
			"page":    q.Page,
			"results": []string{"result1", "result2"},
		})
		return nil
	})

	// 4. Path binding
	router.GET("/users/:id", func(ctx *chain.Context) error {
		// Manual parameter access
		id := ctx.GetParam("id")

		// Or bind to struct
		type PathParam struct {
			ID string `path:"id"`
		}
		var p PathParam
		ctx.BindPath(&p)

		ctx.Json(map[string]string{
			"id_manual": id,
			"id_struct": p.ID,
		})
		return nil
	})

	// 5. Header binding
	router.GET("/headers", func(ctx *chain.Context) error {
		var k APIKey
		if err := ctx.ShouldBindHeader(&k); err != nil {
			ctx.Json(map[string]string{"error": "X-API-Key header required"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}
		ctx.Json(map[string]string{
			"message":    "API key accepted",
			"key_prefix": k.Key[:4] + "...",
		})
		return nil
	})

	// 6. Form binding
	router.POST("/login", func(ctx *chain.Context) error {
		var f LoginForm
		if err := ctx.BindForm(&f); err != nil {
			return err
		}
		ctx.Json(map[string]string{
			"message": "login successful",
			"user":    f.Username,
		})
		return nil
	})

	// 7. Auto-detect binding (Bind — detects Content-Type automatically)
	router.POST("/auto", func(ctx *chain.Context) error {
		var data map[string]any
		if err := ctx.Bind(&data); err != nil {
			return err
		}
		ctx.Json(map[string]any{
			"echo": data,
		})
		return nil
	})

	log.Println("Data binding demo listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
