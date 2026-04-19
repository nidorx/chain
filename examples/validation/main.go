// Package main demonstrates request validation in Chain.
//
// Run with: go run main.go
// Then try:
//
//	# Valid request
//	curl -X POST http://localhost:8080/register \
//	  -H "Content-Type: application/json" \
//	  -d '{"name":"Alice","email":"alice@example.com","password":"secure123","age":30}'
//
//	# Invalid request (missing required fields)
//	curl -X POST http://localhost:8080/register \
//	  -H "Content-Type: application/json" \
//	  -d '{"name":"Al"}'
//
//	# Invalid email
//	curl -X POST http://localhost:8080/register \
//	  -H "Content-Type: application/json" \
//	  -d '{"name":"Alice","email":"not-an-email","password":"secure123"}'
package main

import (
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/nidorx/chain"
)

// RegisterRequest demonstrates various validation tags.
type RegisterRequest struct {
	// Required, between 3 and 100 characters
	Name string `json:"name" binding:"required,min=3,max=100"`

	// Required, must be a valid email format
	Email string `json:"email" binding:"required,email"`

	// Required, between 8 and 128 characters
	Password string `json:"password" binding:"required,min=8,max=128"`

	// Optional, if present must be between 0 and 150
	Age int `json:"age" binding:"omitempty,min=0,max=150"`

	// Optional, must be a valid URL if present
	Website string `json:"website" binding:"omitempty,url"`

	// Required, must be one of the specified values
	Role string `json:"role" binding:"required,oneof=user admin moderator"`

	// Required, must match the password field
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

// UpdateProfile demonstrates partial validation with omitempty.
type UpdateProfile struct {
	Name  string `json:"name"  binding:"omitempty,min=3,max=100"`
	Email string `json:"email" binding:"omitempty,email"`
	Bio   string `json:"bio"   binding:"omitempty,max=500"`
}

// ListQuery demonstrates query parameter validation.
type ListQuery struct {
	Page   int    `query:"page"     binding:"min=1"`
	Limit  int    `query:"limit"    binding:"min=1,max=100"`
	Sort   string `query:"sort"     binding:"omitempty,oneof=name email created_at"`
	Order  string `query:"order"    binding:"omitempty,oneof=asc desc"`
	Search string `query:"search"   binding:"omitempty,max=100"`
}

func main() {
	router := chain.New()

	// Custom error handler that formats validation errors nicely
	router.ErrorHandler = func(ctx *chain.Context, err error) {
		if validationErr, ok := err.(chain.SliceValidationErrors); ok {
			ctx.BadRequest(map[string]any{
				"error":   "Validation failed",
				"details": validationErr.Error(),
			})
			return
		}

		if validationErr, ok := err.(validator.ValidationErrors); ok {
			ctx.BadRequest(map[string]any{
				"error":   "Validation failed",
				"details": validationErr.Error(),
			})
			return
		}

		ctx.Json(map[string]string{
			"error": "Internal Server Error",
		})
		ctx.InternalServerError()
	}

	// ── Registration endpoint ──────────────────────────────────────────
	router.POST("/register", func(ctx *chain.Context) error {
		var req RegisterRequest

		// BindJSON automatically validates based on struct tags
		// If validation fails, it returns an error
		if err := ctx.BindJSON(&req); err != nil {
			return err // handled by ErrorHandler above
		}

		ctx.Status(http.StatusCreated)
		ctx.Json(map[string]any{
			"message": "registration successful",
			"user": map[string]any{
				"name":  req.Name,
				"email": req.Email,
				"role":  req.Role,
			},
		})
		return nil
	})

	// ── Profile update endpoint ────────────────────────────────────────
	router.PUT("/profile/:id", func(ctx *chain.Context) error {
		var req UpdateProfile

		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.Json(map[string]string{"error": err.Error()})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		ctx.Json(map[string]any{
			"message": "profile updated",
			"id":      ctx.GetParam("id"),
			"profile": req,
		})
		return nil
	})

	// ── List endpoint with query validation ────────────────────────────
	router.GET("/items", func(ctx *chain.Context) error {
		var q ListQuery

		// Defaults
		q.Page = 1
		q.Limit = 20
		q.Sort = "created_at"
		q.Order = "desc"

		if err := ctx.BindQuery(&q); err != nil {
			return err
		}

		ctx.Json(map[string]any{
			"page":    q.Page,
			"limit":   q.Limit,
			"sort":    q.Sort,
			"order":   q.Order,
			"search":  q.Search,
			"results": []string{},
		})
		return nil
	})

	// ── Validation with ShouldBind (manual error handling) ─────────────
	router.POST("/manual-validation", func(ctx *chain.Context) error {
		var req RegisterRequest

		// ShouldBindJSON returns the error without setting status code
		if err := ctx.ShouldBindJSON(&req); err != nil {
			// Parse validation errors into a user-friendly format
			ctx.Json(map[string]any{
				"error": "invalid request",
				"fields": map[string]string{
					"name":     "must be 3-100 characters",
					"email":    "must be a valid email",
					"password": "must be at least 8 characters",
					"role":     "must be user, admin, or moderator",
				},
			})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		ctx.Json(map[string]string{"message": "valid"})
		return nil
	})

	log.Println("Validation demo listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
