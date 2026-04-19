// Package main demonstrates error handling in Chain.
//
// Run with: go run main.go
// Then try:
//
//	curl http://localhost:8080/not-found                # 404
//	curl http://localhost:8080/panic                    # 500 (panic recovery)
//	curl http://localhost:8080/api/error                # custom error
//	curl http://localhost:8080/api/validation           # validation error
//	curl -X POST http://localhost:8080/api/multi-error
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/nidorx/chain"
)

// AppError represents a custom application error.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func main() {
	router := chain.New()

	// ── 1. Global error handler ────────────────────────────────────────
	// Catches all unhandled errors and formats the response
	router.ErrorHandler = func(ctx *chain.Context, err error) {
		// Log the full error for debugging
		log.Printf("[ERROR] %v", err)

		// Check if it's our custom AppError
		var appErr *AppError
		if errors.As(err, &appErr) {
			ctx.Status(appErr.Code, map[string]any{
				"error":   appErr.Message,
				"details": appErr.Details,
			})
			return
		}

		// Check if it's a validation error
		if validationErr, ok := err.(chain.SliceValidationErrors); ok {
			ctx.BadRequest(map[string]any{
				"error":   "Validation failed",
				"details": validationErr.Error(),
			})
			return
		}

		// Default: internal server error
		ctx.InternalServerError(map[string]string{
			"error": "Internal Server Error",
		})
	}

	// ── 2. Panic handler ───────────────────────────────────────────────
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, rcv any) {
		stack := debug.Stack()
		log.Printf("[PANIC] %v\n%s", rcv, stack)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error","message":"a panic occurred and was recovered"}`))
	}

	// ── 3. Custom 404 handler ─────────────────────────────────────────
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"error":"not found","path":"%s"}`, r.URL.Path)))
	})

	// ── 4. Custom 405 handler ─────────────────────────────────────────
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(fmt.Sprintf(`{"error":"method not allowed","method":"%s","path":"%s"}`, r.Method, r.URL.Path)))
	})

	// ── Routes ─────────────────────────────────────────────────────────

	// Normal route
	router.GET("/", func(ctx *chain.Context) error {
		ctx.Json(map[string]string{"message": "Error handling demo"})
		return nil
	})

	// Route that triggers a panic (recovered by PanicHandler)
	router.GET("/panic", func(ctx *chain.Context) error {
		panic("intentional panic for demo")
	})

	// Route that returns a custom AppError
	router.GET("/api/error", func(ctx *chain.Context) error {
		return &AppError{
			Code:    http.StatusBadGateway,
			Message: "Upstream service unavailable",
			Details: "The backend service returned a 503 error",
		}
	})

	// Route that returns a validation error
	router.GET("/api/validation", func(ctx *chain.Context) error {
		return chain.SliceValidationErrors{
			errors.New("name is required"),
			errors.New("email must be valid"),
		}
	})

	// Route that returns multiple errors
	router.POST("/api/multi-error", func(ctx *chain.Context) error {
		// Simulate multiple validation errors
		var errs chain.SliceValidationErrors
		errs = append(errs, errors.New("username must be at least 3 characters"))
		errs = append(errs, errors.New("password must be at least 8 characters"))
		errs = append(errs, errors.New("email must be a valid email address"))
		return errs
	})

	// Route that returns a standard Go error
	router.GET("/api/simple-error", func(ctx *chain.Context) error {
		return errors.New("something went wrong")
	})

	// Route group with error handling middleware
	api := router.Group("/api/protected")
	api.Use(func(ctx *chain.Context, next func() error) error {
		err := next()
		if err != nil {
			// Log additional context for API errors
			log.Printf("[API ERROR] path=%s error=%v", ctx.URL().Path, err)
		}
		return err
	})

	api.GET("/data", func(ctx *chain.Context) error {
		// Simulate a database error
		return &AppError{
			Code:    http.StatusServiceUnavailable,
			Message: "Database connection failed",
			Details: "Connection timed out after 30s",
		}
	})

	// Route that demonstrates BeforeSend/AfterSend hooks
	router.GET("/hooks", func(ctx *chain.Context) error {
		// Register a hook that runs before the response is sent
		ctx.BeforeSend(func() {
			log.Println("[BeforeSend] Response is about to be sent")
		})

		// Register a hook that runs after the response is sent
		ctx.AfterSend(func() {
			log.Println("[AfterSend] Response has been sent")
		})

		ctx.Json(map[string]string{"message": "hooks demo"})
		return nil
	})

	log.Println("Error handling demo listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
