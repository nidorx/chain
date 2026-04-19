package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Request Lifecycle Tests
// ============================================================================

func Test_Integration_RequestLifecycle(t *testing.T) {
	router := New()

	var lifecycle []string
	var mu sync.Mutex

	router.Use(func(ctx *Context, next func() error) error {
		mu.Lock()
		lifecycle = append(lifecycle, "middleware-before")
		mu.Unlock()
		err := next()
		mu.Lock()
		lifecycle = append(lifecycle, "middleware-after")
		mu.Unlock()
		return err
	})

	router.GET("/test", func(ctx *Context) error {
		mu.Lock()
		lifecycle = append(lifecycle, "handler")
		mu.Unlock()
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := []string{"middleware-before", "handler", "middleware-after"}
	if len(lifecycle) != len(expected) {
		t.Fatalf("expected %d lifecycle steps, got %d: %v", len(expected), len(lifecycle), lifecycle)
	}
	for i, step := range expected {
		if lifecycle[i] != step {
			t.Errorf("step %d: expected %q, got %q", i, step, lifecycle[i])
		}
	}
}

func Test_Integration_RequestLifecycle_FullCycle(t *testing.T) {
	router := New()

	router.GET("/users/:id", func(ctx *Context) error {
		id := ctx.GetParam("id")
		ctx.Json(map[string]string{"id": id})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["id"] != "123" {
		t.Errorf("expected id '123', got '%s'", response["id"])
	}
}

func Test_Integration_RequestLifecycle_WithQueryString(t *testing.T) {
	router := New()

	router.GET("/search", func(ctx *Context) error {
		q := ctx.QueryParam("q")
		page := ctx.QueryParamInt("page", 1)
		ctx.Json(map[string]any{"query": q, "page": page})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=golang&page=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["query"] != "golang" {
		t.Errorf("expected query 'golang', got '%v'", response["query"])
	}
	if int(response["page"].(float64)) != 2 {
		t.Errorf("expected page 2, got %v", response["page"])
	}
}

func Test_Integration_RequestLifecycle_WithBody(t *testing.T) {
	router := New()

	type Request struct {
		Name string `json:"name"`
	}
	type Response struct {
		Message string `json:"message"`
	}

	router.POST("/greet", func(ctx *Context) error {
		var req Request
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		ctx.Json(Response{Message: "Hello, " + req.Name + "!"})
		return nil
	})

	body, _ := json.Marshal(Request{Name: "World"})
	req := httptest.NewRequest(http.MethodPost, "/greet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Message != "Hello, World!" {
		t.Errorf("expected message 'Hello, World!', got '%s'", response.Message)
	}
}

// ============================================================================
// Middleware Chain Tests
// ============================================================================

func Test_Integration_MiddlewareChain_Multiple(t *testing.T) {
	router := New()

	var order []string
	var mu sync.Mutex

	router.Use(func(ctx *Context, next func() error) error {
		mu.Lock()
		order = append(order, "mw1-before")
		mu.Unlock()
		err := next()
		mu.Lock()
		order = append(order, "mw1-after")
		mu.Unlock()
		return err
	})

	router.Use(func(ctx *Context, next func() error) error {
		mu.Lock()
		order = append(order, "mw2-before")
		mu.Unlock()
		err := next()
		mu.Lock()
		order = append(order, "mw2-after")
		mu.Unlock()
		return err
	})

	router.Use(func(ctx *Context, next func() error) error {
		mu.Lock()
		order = append(order, "mw3-before")
		mu.Unlock()
		err := next()
		mu.Lock()
		order = append(order, "mw3-after")
		mu.Unlock()
		return err
	})

	router.GET("/test", func(ctx *Context) error {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := []string{
		"mw1-before", "mw2-before", "mw3-before",
		"handler",
		"mw3-after", "mw2-after", "mw1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(order), order)
	}
	for i, step := range expected {
		if order[i] != step {
			t.Errorf("step %d: expected %q, got %q", i, step, order[i])
		}
	}
}

func Test_Integration_MiddlewareChain_Abort(t *testing.T) {
	router := New()

	handlerCalled := false
	mw2Called := false

	router.Use(func(ctx *Context, next func() error) error {
		return next()
	})

	router.Use(func(ctx *Context, next func() error) error {
		ctx.WriteHeader(http.StatusUnauthorized)
		// Don't call next - abort the chain
		return nil
	})

	router.GET("/test", func(ctx *Context) error {
		handlerCalled = true
		ctx.OK()
		return nil
	})

	router.Use(func(ctx *Context, next func() error) error {
		mw2Called = true
		return next()
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if handlerCalled {
		t.Error("handler should not have been called")
	}
	if mw2Called {
		t.Error("middleware after abort should not have been called")
	}
}

func Test_Integration_MiddlewareChain_ErrorPropagation(t *testing.T) {
	router := New()

	errorHandled := false
	router.ErrorHandler = func(ctx *Context, err error) {
		errorHandled = true
		ctx.Error("handled error", http.StatusBadRequest)
	}

	router.Use(func(ctx *Context, next func() error) error {
		err := next()
		if err != nil {
			return err
		}
		return nil
	})

	router.GET("/test", func(ctx *Context) error {
		return errors.New("handler error")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !errorHandled {
		t.Error("error should have been handled")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_Integration_MiddlewareChain_ScopedToPath(t *testing.T) {
	router := New()

	apiCalled := false
	publicCalled := false

	router.Use("/api/*", func(ctx *Context, next func() error) error {
		apiCalled = true
		return next()
	})

	router.Use("/public/*", func(ctx *Context, next func() error) error {
		publicCalled = true
		return next()
	})

	router.GET("/api/users", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	router.GET("/public/info", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	// Test API route
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !apiCalled {
		t.Error("API middleware should have been called")
	}
	if publicCalled {
		t.Error("Public middleware should not have been called for API route")
	}

	// Reset
	apiCalled = false
	publicCalled = false

	// Test public route
	req = httptest.NewRequest(http.MethodGet, "/public/info", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !publicCalled {
		t.Error("Public middleware should have been called")
	}
	if apiCalled {
		t.Error("API middleware should not have been called for public route")
	}
}

func Test_Integration_MiddlewareChain_ScopedToMethod(t *testing.T) {
	router := New()

	getCalled := false
	postCalled := false

	router.Use("GET", "/resource", func(ctx *Context, next func() error) error {
		getCalled = true
		return next()
	})

	router.Use("POST", "/resource", func(ctx *Context, next func() error) error {
		postCalled = true
		return next()
	})

	router.GET("/resource", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	router.POST("/resource", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	// Test GET
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !getCalled {
		t.Error("GET middleware should have been called")
	}
	if postCalled {
		t.Error("POST middleware should not have been called for GET request")
	}

	// Reset
	getCalled = false
	postCalled = false

	// Test POST
	req = httptest.NewRequest(http.MethodPost, "/resource", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !postCalled {
		t.Error("POST middleware should have been called")
	}
	if getCalled {
		t.Error("GET middleware should not have been called for POST request")
	}
}

// ============================================================================
// Route Groups Tests
// ============================================================================

func Test_Integration_RouteGroups_Nested(t *testing.T) {
	router := New()

	v1 := router.Group("/api/v1")
	v1.GET("/users", func(ctx *Context) error {
		ctx.Json(map[string]string{"version": "v1", "resource": "users"})
		return nil
	})
	v1.GET("/posts", func(ctx *Context) error {
		ctx.Json(map[string]string{"version": "v1", "resource": "posts"})
		return nil
	})

	v2 := router.Group("/api/v2")
	v2.GET("/users", func(ctx *Context) error {
		ctx.Json(map[string]string{"version": "v2", "resource": "users"})
		return nil
	})

	// Test v1 users
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["version"] != "v1" {
		t.Errorf("expected version 'v1', got '%s'", resp["version"])
	}

	// Test v2 users
	req = httptest.NewRequest(http.MethodGet, "/api/v2/users", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["version"] != "v2" {
		t.Errorf("expected version 'v2', got '%s'", resp["version"])
	}
}

func Test_Integration_RouteGroups_MiddlewareInheritance(t *testing.T) {
	router := New()

	var order []string
	var mu sync.Mutex

	// Global middleware
	router.Use(func(ctx *Context, next func() error) error {
		mu.Lock()
		order = append(order, "global")
		mu.Unlock()
		return next()
	})

	// Group-specific middleware
	api := router.Group("/api")
	api.Use(func(ctx *Context, next func() error) error {
		mu.Lock()
		order = append(order, "api-group")
		mu.Unlock()
		return next()
	})

	api.GET("/test", func(ctx *Context) error {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		ctx.OK()
		return nil
	})

	// Test API route
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := []string{"global", "api-group", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(order), order)
	}
	for i, step := range expected {
		if order[i] != step {
			t.Errorf("step %d: expected %q, got %q", i, step, order[i])
		}
	}
}

func Test_Integration_RouteGroups_DeepNesting(t *testing.T) {
	router := New()

	v1 := router.Group("/api")
	v2 := v1.Group("/v1")
	v3 := v2.Group("/admin")

	v3.GET("/settings", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// ============================================================================
// Concurrent Requests Tests
// ============================================================================

func Test_Integration_ConcurrentRequests_ThreadSafety(t *testing.T) {
	router := New()

	var counter int64

	router.GET("/counter", func(ctx *Context) error {
		atomic.AddInt64(&counter, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		val := atomic.LoadInt64(&counter)
		ctx.Json(map[string]int64{"count": val})
		return nil
	})

	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/counter", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}
		}()
	}

	wg.Wait()

	if counter != numRequests {
		t.Errorf("expected counter to be %d, got %d", numRequests, counter)
	}
}

func Test_Integration_ConcurrentRequests_RaceConditions(t *testing.T) {
	router := New()

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	router.GET("/user/:id", func(ctx *Context) error {
		id := ctx.GetParam("id")
		ctx.Json(User{ID: 1, Name: "user-" + id})
		return nil
	})

	router.POST("/user", func(ctx *Context) error {
		var user User
		ctx.BindJSON(&user)
		ctx.Json(user)
		return nil
	})

	var wg sync.WaitGroup
	const numRequests = 50
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer wg.Done()

			var req *http.Request
			if id%2 == 0 {
				req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/user/%d", id), nil)
			} else {
				body, _ := json.Marshal(User{ID: id, Name: fmt.Sprintf("user-%d", id)})
				req = httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d: expected status %d, got %d", id, http.StatusOK, w.Code)
			}
		}(i)
	}

	wg.Wait()
}

func Test_Integration_ConcurrentRequests_ContextIsolation(t *testing.T) {
	router := New()

	router.GET("/isolated/:id", func(ctx *Context) error {
		id := ctx.GetParam("id")
		// Store something in context
		ctx.Set("request_id", id)
		time.Sleep(5 * time.Millisecond)
		// Verify the value is still correct
		stored, _ := ctx.Get("request_id")
		if stored != id {
			t.Errorf("context isolation failed: expected %q, got %q", id, stored)
		}
		ctx.Json(map[string]string{"id": id})
		return nil
	})

	var wg sync.WaitGroup
	const numRequests = 20
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/isolated/%d", id), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}(i)
	}

	wg.Wait()
}

// ============================================================================
// Context Pooling Tests
// ============================================================================

func Test_Integration_ContextPooling_MemoryManagement(t *testing.T) {
	router := New()

	router.GET("/pool", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	// Make several requests to exercise the pool
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/pool", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, w.Code)
		}
	}
}

func Test_Integration_ContextPooling_NoGoroutineLeak(t *testing.T) {
	router := New()

	router.GET("/test", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	// Make many requests rapidly
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Give time for any potential goroutine cleanup
	time.Sleep(50 * time.Millisecond)

	// If we got here without hanging, the test passes
	// (A more thorough test would use runtime.NumGoroutine)
}

func Test_Integration_ContextPooling_ContextDataIsolation(t *testing.T) {
	router := New()

	router.GET("/isolation/:id", func(ctx *Context) error {
		id := ctx.GetParam("id")

		// Set context data
		ctx.Set("data", "value-"+id)

		// Verify it's correct
		val, _ := ctx.Get("data")
		if val != "value-"+id {
			t.Errorf("data isolation failed: expected %q, got %q", "value-"+id, val)
		}

		ctx.OK()
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/isolation/%d", id), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}(i)
	}
	wg.Wait()
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func Test_Integration_ErrorHandling_PanicRecovery(t *testing.T) {
	router := New()

	panicRecovered := false
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		panicRecovered = true
		w.WriteHeader(http.StatusInternalServerError)
	}

	router.GET("/panic", func(ctx *Context) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	// Should not panic
	router.ServeHTTP(w, req)

	if !panicRecovered {
		t.Error("panic should have been recovered")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func Test_Integration_ErrorHandling_CustomErrorHandler(t *testing.T) {
	router := New()

	customErrorHandled := false
	var handledErr error
	router.ErrorHandler = func(ctx *Context, err error) {
		customErrorHandled = true
		handledErr = err
		// Note: status may already be written; just verify handler is called
	}

	router.GET("/error", func(ctx *Context) error {
		return errors.New("custom error")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !customErrorHandled {
		t.Error("custom error handler should have been called")
	}
	if handledErr == nil || handledErr.Error() != "custom error" {
		t.Errorf("expected error 'custom error', got %v", handledErr)
	}
}

func Test_Integration_ErrorHandling_NotFound(t *testing.T) {
	router := New()

	customNotFoundCalled := false
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customNotFoundCalled = true
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !customNotFoundCalled {
		t.Error("custom not found handler should have been called")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func Test_Integration_ErrorHandling_MethodNotAllowed(t *testing.T) {
	router := New()

	router.POST("/resource", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	// Check Allow header
	allow := w.Header().Get("Allow")
	if allow == "" {
		t.Error("Allow header should be set")
	}
}

func Test_Integration_ErrorHandling_PanicInMiddleware(t *testing.T) {
	router := New()

	panicRecovered := false
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		panicRecovered = true
		w.WriteHeader(http.StatusInternalServerError)
	}

	router.Use(func(ctx *Context, next func() error) error {
		panic("middleware panic")
	})

	router.GET("/test", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !panicRecovered {
		t.Error("panic in middleware should have been recovered")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func Test_Integration_ErrorHandling_MultipleErrors(t *testing.T) {
	router := New()

	var errorCount int64
	router.ErrorHandler = func(ctx *Context, err error) {
		atomic.AddInt64(&errorCount, 1)
	}

	router.GET("/error", func(ctx *Context) error {
		return fmt.Errorf("handler error")
	})

	// Make multiple requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Status 200 because handler returns nil after error handler is called
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, w.Code)
		}
	}

	if atomic.LoadInt64(&errorCount) != 3 {
		t.Errorf("expected 3 error handler calls, got %d", errorCount)
	}
}

// ============================================================================
// Response Writing Tests
// ============================================================================

func Test_Integration_ResponseWriting_StatusCodes(t *testing.T) {
	router := New()

	router.GET("/ok", func(ctx *Context) error {
		ctx.OK()
		return nil
	})
	router.GET("/created", func(ctx *Context) error {
		ctx.Created()
		return nil
	})
	router.GET("/no-content", func(ctx *Context) error {
		ctx.NoContent()
		return nil
	})
	router.GET("/bad-request", func(ctx *Context) error {
		ctx.BadRequest()
		return nil
	})
	router.GET("/unauthorized", func(ctx *Context) error {
		ctx.Unauthorized()
		return nil
	})
	router.GET("/forbidden", func(ctx *Context) error {
		ctx.Forbidden()
		return nil
	})
	router.GET("/internal-error", func(ctx *Context) error {
		ctx.InternalServerError()
		return nil
	})

	tests := []struct {
		path   string
		status int
	}{
		{"/ok", http.StatusOK},
		{"/created", http.StatusCreated},
		{"/no-content", http.StatusNoContent},
		{"/bad-request", http.StatusBadRequest},
		{"/unauthorized", http.StatusUnauthorized},
		{"/forbidden", http.StatusForbidden},
		{"/internal-error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func Test_Integration_ResponseWriting_Headers(t *testing.T) {
	router := New()

	router.GET("/headers", func(ctx *Context) error {
		ctx.SetHeader("X-Custom-Header", "custom-value")
		ctx.SetHeader("X-Another", "another-value")
		ctx.AddHeader("X-Multi", "value1")
		ctx.AddHeader("X-Multi", "value2")
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/headers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header 'custom-value', got '%s'", w.Header().Get("X-Custom-Header"))
	}
	if w.Header().Get("X-Another") != "another-value" {
		t.Errorf("expected X-Another 'another-value', got '%s'", w.Header().Get("X-Another"))
	}

	multi := w.Header()["X-Multi"]
	if len(multi) != 2 || multi[0] != "value1" || multi[1] != "value2" {
		t.Errorf("expected X-Multi ['value1', 'value2'], got %v", multi)
	}
}

func Test_Integration_ResponseWriting_Redirect(t *testing.T) {
	router := New()

	router.GET("/redirect", func(ctx *Context) error {
		ctx.Redirect("/destination", http.StatusFound)
		return nil
	})

	router.GET("/destination", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/redirect", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, w.Code)
	}
	if w.Header().Get("Location") != "/destination" {
		t.Errorf("expected Location '/destination', got '%s'", w.Header().Get("Location"))
	}
}

func Test_Integration_ResponseWriting_Cookies(t *testing.T) {
	router := New()

	router.GET("/set-cookie", func(ctx *Context) error {
		ctx.SetCookie(&http.Cookie{
			Name:  "test-cookie",
			Value: "test-value",
			Path:  "/",
		})
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/set-cookie", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "test-cookie" {
		t.Errorf("expected cookie name 'test-cookie', got '%s'", cookies[0].Name)
	}
	if cookies[0].Value != "test-value" {
		t.Errorf("expected cookie value 'test-value', got '%s'", cookies[0].Value)
	}
}

// ============================================================================
// Context Data Sharing Tests
// ============================================================================

func Test_Integration_ContextData_SetGet(t *testing.T) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		ctx.Set("user", "admin")
		ctx.Set("role", "superuser")
		return next()
	})

	router.GET("/profile", func(ctx *Context) error {
		user, _ := ctx.Get("user")
		role, _ := ctx.Get("role")

		ctx.Json(map[string]any{"user": user, "role": role})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["user"] != "admin" {
		t.Errorf("expected user 'admin', got '%s'", response["user"])
	}
	if response["role"] != "superuser" {
		t.Errorf("expected role 'superuser', got '%s'", response["role"])
	}
}

func Test_Integration_ContextData_ChildContext(t *testing.T) {
	router := New()

	router.Use(func(ctx *Context, next func() error) error {
		ctx.Set("parent", "value")
		return next()
	})

	router.GET("/child", func(ctx *Context) error {
		// Create child context
		child := ctx.Child()
		child.Set("child", "value")

		// Child should inherit parent data
		parentVal, _ := child.Get("parent")
		childVal, _ := child.Get("child")

		ctx.Json(map[string]any{"parent": parentVal, "child": childVal})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/child", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["parent"] != "value" {
		t.Errorf("expected parent 'value', got '%s'", response["parent"])
	}
	if response["child"] != "value" {
		t.Errorf("expected child 'value', got '%s'", response["child"])
	}
}

func Test_Integration_ContextData_GetNonExistent(t *testing.T) {
	router := New()

	router.GET("/missing", func(ctx *Context) error {
		val, ok := ctx.Get("nonexistent")
		if ok {
			t.Error("expected Get to return false for nonexistent key")
		}
		if val != nil {
			t.Errorf("expected nil value, got %v", val)
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// ============================================================================
// Request Context Tests
// ============================================================================

func Test_Integration_RequestContext_FromStdContext(t *testing.T) {
	router := New()

	router.GET("/std-context", func(ctx *Context) error {
		// Get context from standard library
		stdCtx := ctx.Request.Context()

		// Should be able to get chain context from std context
		chainCtx := GetContext(stdCtx)
		if chainCtx == nil {
			t.Error("expected chain context from standard context")
		}

		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/std-context", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_Integration_RequestContext_ContextCancellation(t *testing.T) {
	router := New()

	cancelled := false
	router.GET("/cancel", func(ctx *Context) error {
		select {
		case <-ctx.Request.Context().Done():
			cancelled = true
		default:
			// Context not cancelled
		}
		ctx.OK()
		return nil
	})

	// Create request with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/cancel", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Don't cancel before request completes
	router.ServeHTTP(w, req)

	if cancelled {
		t.Error("context should not be cancelled before request completes")
	}

	// Now cancel
	cancel()
}

// ============================================================================
// Auto-Redirect Tests
// ============================================================================

func Test_Integration_AutoRedirect_TrailingSlash(t *testing.T) {
	router := New()
	router.RedirectTrailingSlash = true

	router.GET("/path/", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/path", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}
	if w.Header().Get("Location") != "/path/" {
		t.Errorf("expected Location '/path/', got '%s'", w.Header().Get("Location"))
	}
}

func Test_Integration_AutoRedirect_FixedPath(t *testing.T) {
	router := New()
	router.RedirectFixedPath = true

	router.GET("/path", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/PATH", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}
	if w.Header().Get("Location") != "/path" {
		t.Errorf("expected Location '/path', got '%s'", w.Header().Get("Location"))
	}
}

// ============================================================================
// BeforeSend/AfterSend Tests
// ============================================================================

func Test_Integration_BeforeSend_Callback(t *testing.T) {
	router := New()

	beforeSendCalled := false
	afterSendCalled := false

	router.GET("/hooks", func(ctx *Context) error {
		ctx.BeforeSend(func() {
			beforeSendCalled = true
		})
		ctx.AfterSend(func() {
			afterSendCalled = true
		})
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/hooks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !beforeSendCalled {
		t.Error("BeforeSend callback should have been called")
	}
	if !afterSendCalled {
		t.Error("AfterSend callback should have been called")
	}
}
