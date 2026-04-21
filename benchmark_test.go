// Copyright 2026 Chain Framework Contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Benchmark Suite
// ============================================================================
// This file contains comprehensive benchmarks for the Chain framework,
// covering all critical performance paths as specified in the evolution roadmap.
//
// Benchmarks are organized by category:
// 1. Route Lookup (static, parameter, wildcard)
// 2. Middleware Execution
// 3. Context Creation & Pooling
// 4. Data Binding (JSON, Form, Query)
// 5. Full Request Cycle
//
// To run all benchmarks:
//   go test -bench=. -benchmem
//
// To run specific benchmark:
//   go test -bench=BenchmarkRouter_StaticRoute -benchmem
// ============================================================================

// benchHandler creates a no-op handler for benchmarks
func benchHandler(name string) func(ctx *Context) error {
	return func(ctx *Context) error {
		return nil
	}
}

// ============================================================================
// 1. Route Lookup Benchmarks
// ============================================================================

// BenchmarkRouter_StaticRoute_Lookup benchmarks static route lookup performance
// This is the most common case and should be extremely fast (O(1) map lookup)
func BenchmarkRouter_StaticRoute_Lookup(b *testing.B) {
	router := New()

	// Register static routes
	routes := []string{
		"/",
		"/api",
		"/api/users",
		"/api/users/list",
		"/api/users/create",
		"/api/posts",
		"/api/posts/list",
		"/api/posts/create",
		"/health",
		"/status",
	}

	for _, route := range routes {
		router.GET(route, benchHandler(route))
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Lookup different static routes
	paths := []string{"/", "/api", "/api/users", "/api/posts", "/health"}
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			handle, _ := router.Lookup(http.MethodGet, path)
			if handle == nil {
				b.Fatalf("expected handle for path %s", path)
			}
		}
	}
}

// BenchmarkRouter_StaticRoute_100 measures lookup performance with 100 routes
func BenchmarkRouter_StaticRoute_100(b *testing.B) {
	router := New()

	// Register 100 static routes with unique paths
	for i := 0; i < 100; i++ {
		path := "/api/v1/res" + string(rune('A'+i/26)) + string(rune('a'+i%26))
		router.GET(path, benchHandler(path))
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Lookup a path that actually exists in the router
	for i := 0; i < b.N; i++ {
		handle, _ := router.Lookup(http.MethodGet, "/api/v1/resAa")
		if handle == nil {
			b.Fatal("expected handle")
		}
	}
}

// BenchmarkRouter_StaticRoute_1000 measures lookup performance with 1000 routes
func BenchmarkRouter_StaticRoute_1000(b *testing.B) {
	router := New()

	// Register 1000 static routes with unique paths
	for i := 0; i < 1000; i++ {
		path := "/api/v1/r" + string(rune('A'+i/676)) + string(rune('a'+(i/26)%26)) + string(rune('a'+i%26))
		router.GET(path, benchHandler(path))
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Lookup a path that actually exists in the router
	for i := 0; i < b.N; i++ {
		handle, _ := router.Lookup(http.MethodGet, "/api/v1/rAaa")
		if handle == nil {
			b.Fatal("expected handle")
		}
	}
}

// BenchmarkRouter_ParameterRoute_Lookup benchmarks parameterized route lookup
func BenchmarkRouter_ParameterRoute_Lookup(b *testing.B) {
	router := New()

	// Register parameterized routes
	routes := []string{
		"/users/:id",
		"/users/:id/profile",
		"/users/:id/posts",
		"/posts/:postId",
		"/posts/:postId/comments",
		"/posts/:postId/comments/:commentId",
		"/api/:version/users/:id",
		"/api/:version/posts/:postId",
	}

	for _, route := range routes {
		router.GET(route, benchHandler(route))
	}

	b.ReportAllocs()
	b.ResetTimer()

	paths := []string{
		"/users/123",
		"/users/123/profile",
		"/posts/456/comments/789",
		"/api/v1/users/123",
	}

	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			handle, ctx := router.Lookup(http.MethodGet, path)
			if handle == nil {
				b.Fatalf("expected handle for path %s", path)
			}
			_ = ctx
		}
	}
}

// BenchmarkRouter_ParameterRoute_100 measures with 100 parameterized routes
func BenchmarkRouter_ParameterRoute_100(b *testing.B) {
	router := New()

	// Register 100 parameterized routes with unique patterns
	for i := 0; i < 100; i++ {
		path := "/api/v1/r" + string(rune('A'+i/26)) + string(rune('a'+i%26)) + "/:id"
		router.GET(path, benchHandler(path))
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Lookup a path that matches one of the registered routes
	for i := 0; i < b.N; i++ {
		handle, ctx := router.Lookup(http.MethodGet, "/api/v1/rAa/123")
		if handle == nil {
			b.Fatal("expected handle")
		}
		_ = ctx
	}
}

// BenchmarkRouter_WildcardRoute_Lookup benchmarks wildcard route lookup
func BenchmarkRouter_WildcardRoute_Lookup(b *testing.B) {
	router := New()

	// Register wildcard routes
	routes := []string{
		"/static/*filepath",
		"/assets/*filepath",
		"/files/*filepath",
		"/downloads/*filepath",
		"/docs/*filepath",
	}

	for _, route := range routes {
		router.GET(route, benchHandler(route))
	}

	b.ReportAllocs()
	b.ResetTimer()

	paths := []string{
		"/static/js/app.js",
		"/assets/css/style.css",
		"/files/documents/report.pdf",
		"/downloads/images/photo.jpg",
	}

	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			handle, ctx := router.Lookup(http.MethodGet, path)
			if handle == nil {
				b.Fatalf("expected handle for path %s", path)
			}
			_ = ctx
		}
	}
}

// BenchmarkRouter_WildcardRoute_DeepPath benchmarks deep path matching with wildcards
func BenchmarkRouter_WildcardRoute_DeepPath(b *testing.B) {
	router := New()

	router.GET("/*filepath", benchHandler("root"))
	router.GET("/static/*filepath", benchHandler("static"))
	router.GET("/static/js/*filepath", benchHandler("static-js"))

	b.ReportAllocs()
	b.ResetTimer()

	// Deep nested path
	path := "/static/js/vendor/jquery/3.6.0/jquery.min.js"
	for i := 0; i < b.N; i++ {
		handle, ctx := router.Lookup(http.MethodGet, path)
		if handle == nil {
			b.Fatal("expected handle")
		}
		_ = ctx
	}
}

// BenchmarkRouter_MixedRoutes compares performance with mixed route types
func BenchmarkRouter_MixedRoutes(b *testing.B) {
	router := New()

	// Mix of static, parameter, and wildcard routes
	staticRoutes := []string{"/", "/api", "/health", "/status"}
	paramRoutes := []string{"/users/:id", "/posts/:postId", "/comments/:commentId"}
	wildcardRoutes := []string{"/static/*filepath", "/files/*filepath"}

	for _, route := range staticRoutes {
		router.GET(route, benchHandler(route))
	}
	for _, route := range paramRoutes {
		router.GET(route, benchHandler(route))
	}
	for _, route := range wildcardRoutes {
		router.GET(route, benchHandler(route))
	}

	b.ReportAllocs()
	b.ResetTimer()

	paths := []string{
		"/",
		"/api",
		"/users/123",
		"/posts/456",
		"/static/js/app.js",
		"/files/docs/readme.md",
	}

	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			handle, _ := router.Lookup(http.MethodGet, path)
			if handle == nil {
				b.Fatalf("expected handle for path %s", path)
			}
		}
	}
}

// ============================================================================
// 2. Middleware Execution Benchmarks
// ============================================================================

// BenchmarkMiddleware_NoMiddleware benchmarks request handling without any middleware
func BenchmarkMiddleware_NoMiddleware(b *testing.B) {
	router := New()

	router.GET("/api/users", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkMiddleware_SingleMiddleware benchmarks with single global middleware
func BenchmarkMiddleware_SingleMiddleware(b *testing.B) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/users", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkMiddleware_MultipleMiddleware benchmarks with multiple middlewares
func BenchmarkMiddleware_MultipleMiddleware(b *testing.B) {
	router := New()

	// Add 5 middlewares
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/users", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkMiddleware_ManyMiddleware benchmarks with 10 middlewares
func BenchmarkMiddleware_ManyMiddleware(b *testing.B) {
	router := New()

	// Add 10 middlewares
	for j := 0; j < 10; j++ {
		router.Use(func(ctx *Context, next func() error) error {
			return next()
		})
	}

	router.GET("/api/users", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkMiddleware_PathScoped benchmarks path-scoped middleware
func BenchmarkMiddleware_PathScoped(b *testing.B) {
	router := New()

	// Global middleware
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	// Path-scoped middleware (should only match /api/*)
	router.Use("/api", func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/users", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkMiddleware_MethodScoped benchmarks method-scoped middleware
func BenchmarkMiddleware_MethodScoped(b *testing.B) {
	router := New()

	// Method-scoped middleware (should only match GET requests)
	router.Use("GET", "/api", func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/users", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// ============================================================================
// 3. Context Creation & Pooling Benchmarks
// ============================================================================

// BenchmarkContext_Creation benchmarks context creation without pooling
func BenchmarkContext_Creation(b *testing.B) {
	router := New()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := router.poolGetContext(req, w, "")
		router.poolPutContext(ctx)
	}
}

// BenchmarkContext_PoolGetPut benchmarks context pool get/put cycle
func BenchmarkContext_PoolGetPut(b *testing.B) {
	router := New()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Get from pool
		ctx := router.poolGetContext(req, w, "")

		// Simulate some work
		ctx.path = "/test"
		ctx.paramCount = 0

		// Return to pool
		router.poolPutContext(ctx)
	}
}

// BenchmarkContext_PoolRecycling benchmarks pool recycling under concurrent access
func BenchmarkContext_PoolRecycling(b *testing.B) {
	router := New()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			ctx := router.poolGetContext(req, w, "")
			ctx.path = "/test"
			router.poolPutContext(ctx)
		}
	})
}

// BenchmarkContext_ParameterExtraction benchmarks parameter extraction performance
func BenchmarkContext_ParameterExtraction(b *testing.B) {
	router := New()

	router.GET("/users/:id/posts/:postId/comments/:commentId", func(ctx *Context) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handle, ctx := router.Lookup(http.MethodGet, "/users/123/posts/456/comments/789")
		if handle == nil {
			b.Fatal("expected handle")
		}

		// Extract parameters
		_ = ctx.GetParam("id")
		_ = ctx.GetParam("postId")
		_ = ctx.GetParam("commentId")
	}
}

// BenchmarkContext_DataStore benchmarks context data storage operations
func BenchmarkContext_DataStore(b *testing.B) {
	router := New()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		ctx := router.poolGetContext(req, w, "")

		// Set and get data
		ctx.Set("user_id", 123)
		ctx.Set("username", "testuser")
		ctx.Set("role", "admin")

		if _, ok := ctx.Get("user_id"); !ok {
			b.Fatal("expected to get user_id")
		}

		router.poolPutContext(ctx)
	}
}

// ============================================================================
// 4. Data Binding Benchmarks
// ============================================================================

// BenchmarkBinding_JSON benchmarks JSON binding performance
func BenchmarkBinding_JSON(b *testing.B) {
	router := New()

	type User struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Age     int    `json:"age"`
		Address string `json:"address"`
	}

	router.POST("/users", func(ctx *Context) error {
		var user User
		return ctx.BindJSON(&user)
	})

	body := User{
		Name:    "John Doe",
		Email:   "john@example.com",
		Age:     30,
		Address: "123 Main St",
	}
	bodyBytes, _ := json.Marshal(body)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBinding_JSON_Small benchmarks small JSON binding
func BenchmarkBinding_JSON_Small(b *testing.B) {
	router := New()

	type Simple struct {
		ID int `json:"id"`
	}

	router.POST("/data", func(ctx *Context) error {
		var data Simple
		return ctx.BindJSON(&data)
	})

	body := Simple{ID: 123}
	bodyBytes, _ := json.Marshal(body)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/data", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBinding_JSON_Large benchmarks large JSON binding
func BenchmarkBinding_JSON_Large(b *testing.B) {
	router := New()

	type LargeStruct struct {
		Field1  string `json:"field1"`
		Field2  string `json:"field2"`
		Field3  string `json:"field3"`
		Field4  string `json:"field4"`
		Field5  string `json:"field5"`
		Field6  string `json:"field6"`
		Field7  string `json:"field7"`
		Field8  string `json:"field8"`
		Field9  string `json:"field9"`
		Field10 string `json:"field10"`
	}

	router.POST("/data", func(ctx *Context) error {
		var data LargeStruct
		return ctx.BindJSON(&data)
	})

	body := LargeStruct{
		Field1: "value1", Field2: "value2", Field3: "value3",
		Field4: "value4", Field5: "value5", Field6: "value6",
		Field7: "value7", Field8: "value8", Field9: "value9",
		Field10: "value10",
	}
	bodyBytes, _ := json.Marshal(body)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/data", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBinding_Query benchmarks query parameter binding
func BenchmarkBinding_Query(b *testing.B) {
	router := New()

	type QueryParams struct {
		Page  int    `query:"page"`
		Limit int    `query:"limit"`
		Sort  string `query:"sort"`
	}

	router.GET("/items", func(ctx *Context) error {
		var params QueryParams
		return ctx.BindQuery(&params)
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/items?page=1&limit=10&sort=name", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBinding_Form benchmarks form data binding
func BenchmarkBinding_Form(b *testing.B) {
	router := New()

	type FormData struct {
		Name    string `form:"name"`
		Email   string `form:"email"`
		Message string `form:"message"`
	}

	router.POST("/submit", func(ctx *Context) error {
		var data FormData
		return ctx.BindForm(&data)
	})

	formData := "name=John&email=john@example.com&message=Hello+World"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader([]byte(formData)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBinding_Path benchmarks path parameter binding
func BenchmarkBinding_Path(b *testing.B) {
	router := New()

	router.GET("/users/:id/posts/:postId", func(ctx *Context) error {
		_ = ctx.GetParam("id")
		_ = ctx.GetParam("postId")
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBinding_Header benchmarks header binding
func BenchmarkBinding_Header(b *testing.B) {
	router := New()

	type Headers struct {
		ContentType string `header:"Content-Type"`
		Auth        string `header:"Authorization"`
		UserAgent   string `header:"User-Agent"`
	}

	router.GET("/api", func(ctx *Context) error {
		var headers Headers
		return ctx.BindHeader(&headers)
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")
		req.Header.Set("User-Agent", "BenchmarkClient/1.0")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// ============================================================================
// 5. Full Request Cycle Benchmarks
// ============================================================================

// BenchmarkFullRequest_StaticRoute benchmarks complete request lifecycle for static routes
func BenchmarkFullRequest_StaticRoute(b *testing.B) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/users", func(ctx *Context) error {
		ctx.Json(map[string]string{"status": "ok"})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_ParameterRoute benchmarks complete request lifecycle for parameterized routes
func BenchmarkFullRequest_ParameterRoute(b *testing.B) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/users/:id", func(ctx *Context) error {
		id := ctx.GetParam("id")
		ctx.Json(map[string]string{"id": id})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_WildcardRoute benchmarks complete request lifecycle for wildcard routes
func BenchmarkFullRequest_WildcardRoute(b *testing.B) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/static/*filepath", func(ctx *Context) error {
		filepath := ctx.GetParam("filepath")
		ctx.Json(map[string]string{"filepath": filepath})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/static/js/app.js", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_WithMiddleware benchmarks request with multiple middlewares
func BenchmarkFullRequest_WithMiddleware(b *testing.B) {
	router := New()

	// Add multiple middlewares
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})
	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/data", func(ctx *Context) error {
		ctx.Json(map[string]string{"data": "value"})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_JSONBinding benchmarks request with JSON binding
func BenchmarkFullRequest_JSONBinding(b *testing.B) {
	router := New()

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	router.POST("/users", func(ctx *Context) error {
		var user User
		if err := ctx.BindJSON(&user); err != nil {
			return err
		}
		ctx.Json(map[string]string{"status": "created"})
		return nil
	})

	body := User{Name: "John", Email: "john@example.com"}
	bodyBytes, _ := json.Marshal(body)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_RouteGroups benchmarks route groups performance
func BenchmarkFullRequest_RouteGroups(b *testing.B) {
	router := New()

	// Create route groups
	api := router.Group("/api")
	v1 := api.Group("/v1")

	v1.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	v1.GET("/users", func(ctx *Context) error {
		ctx.Json(map[string]string{"version": "v1"})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_ErrorHandling benchmarks error handling performance
func BenchmarkFullRequest_ErrorHandling(b *testing.B) {
	router := New()

	router.ErrorHandler = func(ctx *Context, err error) {
		ctx.Json(map[string]string{"error": err.Error()})
		ctx.Status(http.StatusInternalServerError)
	}

	router.GET("/error", func(ctx *Context) error {
		return errors.New("simulated error") // Simulated error for benchmark
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequest_Concurrent benchmarks concurrent request handling
func BenchmarkFullRequest_Concurrent(b *testing.B) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.GET("/api/data", func(ctx *Context) error {
		ctx.Json(map[string]string{"data": "value"})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// ============================================================================
// 6. Route Registration Benchmarks
// ============================================================================

// BenchmarkRouteRegistration_Static benchmarks static route registration
func BenchmarkRouteRegistration_Static(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		router := New()
		b.StartTimer()

		// Register 50 static routes with unique paths
		for j := 0; j < 50; j++ {
			path := "/api/v1/r" + string(rune('A'+j/26)) + string(rune('a'+j%26)) + string(rune('0'+(i+j)%10))
			router.GET(path, benchHandler(path))
		}
	}
}

// BenchmarkRouteRegistration_Parameter benchmarks parameterized route registration
func BenchmarkRouteRegistration_Parameter(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		router := New()
		b.StartTimer()

		// Register 50 parameterized routes with unique patterns
		for j := 0; j < 50; j++ {
			path := "/api/v1/r" + string(rune('A'+j/26)) + string(rune('a'+j%26)) + string(rune('0'+(i+j)%10)) + "/:id"
			router.GET(path, benchHandler(path))
		}
	}
}

// BenchmarkRouteRegistration_Wildcard benchmarks wildcard route registration
func BenchmarkRouteRegistration_Wildcard(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		router := New()
		b.StartTimer()

		// Register 20 wildcard routes with unique parent paths
		// Note: Wildcards can only be registered once per parent path
		for j := 0; j < 20; j++ {
			path := "/static/dir" + string(rune('A'+j)) + "/*filepath"
			router.GET(path, benchHandler(path))
		}
	}
}

// ============================================================================
// 7. Response Writing Benchmarks
// ============================================================================

// BenchmarkResponse_JSON benchmarks JSON response writing
func BenchmarkResponse_JSON(b *testing.B) {
	router := New()

	router.GET("/api/data", func(ctx *Context) error {
		ctx.Json(map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkResponse_Status benchmarks status code setting
func BenchmarkResponse_Status(b *testing.B) {
	router := New()

	router.GET("/api/data", func(ctx *Context) error {
		ctx.Status(http.StatusCreated)
		ctx.Json(map[string]string{"status": "created"})
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
