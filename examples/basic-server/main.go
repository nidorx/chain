// Package main demonstrates a basic Chain HTTP server.
//
// Run with: go run main.go
// Then visit: http://localhost:8080
package main

import (
	"log"
	"net/http"

	"github.com/nidorx/chain"
)

func main() {
	// Create a new router with default settings
	router := chain.New()

	// Root handler — returns a JSON response
	router.GET("/", func(ctx *chain.Context) error {
		ctx.Json(map[string]string{
			"message":   "Welcome to Chain!",
			"version":   "1.0.0",
			"docs":      "https://pkg.go.dev/github.com/nidorx/chain",
			"repository": "https://github.com/nidorx/chain",
		})
		return nil
	})

	// Health check endpoint
	router.GET("/health", func(ctx *chain.Context) error {
		ctx.Json(map[string]string{"status": "ok"})
		return nil
	})

	// Simple text response
	router.GET("/ping", func(ctx *chain.Context) error {
		ctx.Write([]byte("pong"))
		return nil
	})

	// Start the server
	log.Println("Basic server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
