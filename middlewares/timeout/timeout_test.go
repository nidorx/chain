package timeout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nidorx/chain"
)

// ============================================================================
// Basic Functionality Tests
// ============================================================================

func Test_New_CompletesBeforeTimeout(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: 200 * time.Millisecond,
	}))

	router.GET("/fast", func(ctx *chain.Context) error {
		time.Sleep(50 * time.Millisecond)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_New_TimeoutExceeded(t *testing.T) {
	router := chain.New()

	handlerCompleted := false
	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	router.GET("/slow", func(ctx *chain.Context) error {
		time.Sleep(100 * time.Millisecond)
		handlerCompleted = true
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	// Handler may still complete (non-cooperative), but response is blocked
	if handlerCompleted {
		t.Log("handler completed but response was blocked (expected for non-cooperative handlers)")
	}
}

func Test_New_ContextCancelledOnTimeout(t *testing.T) {
	router := chain.New()

	var contextCancelled bool

	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	router.GET("/slow", func(ctx *chain.Context) error {
		// Check if context gets cancelled
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Request.Context().Done():
				contextCancelled = true
				return ctx.Request.Context().Err()
			case <-time.After(10 * time.Millisecond):
				// Keep waiting
			}
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	if !contextCancelled {
		t.Error("context should have been cancelled")
	}

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// ============================================================================
// Configuration Tests
// ============================================================================

func Test_New_ZeroTimeout(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: 0,
	}))

	handlerCompleted := false
	router.GET("/test", func(ctx *chain.Context) error {
		time.Sleep(10 * time.Millisecond)
		handlerCompleted = true
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !handlerCompleted {
		t.Error("handler should have completed")
	}
}

func Test_New_NegativeTimeout(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: -1 * time.Second,
	}))

	handlerCompleted := false
	router.GET("/test", func(ctx *chain.Context) error {
		time.Sleep(10 * time.Millisecond)
		handlerCompleted = true
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !handlerCompleted {
		t.Error("handler should have completed")
	}
}

func Test_New_CustomStatusCode(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout:    50 * time.Millisecond,
		StatusCode: http.StatusGatewayTimeout,
	}))

	router.GET("/slow", func(ctx *chain.Context) error {
		time.Sleep(100 * time.Millisecond)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, w.Code)
	}
}

func Test_New_CustomErrorHandler(t *testing.T) {
	router := chain.New()

	customHandlerCalled := false
	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
		ErrorHandler: func(ctx *chain.Context) {
			customHandlerCalled = true
			ctx.Json(map[string]string{"error": "custom timeout"})
		},
	}))

	router.GET("/slow", func(ctx *chain.Context) error {
		time.Sleep(100 * time.Millisecond)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !customHandlerCalled {
		t.Error("custom error handler should have been called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d (custom handler sets 200), got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != `{"error":"custom timeout"}` {
		t.Errorf("expected custom error response, got %q", w.Body.String())
	}
}

func Test_New_IncludeTimeoutHeader(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout:              5 * time.Second,
		IncludeTimeoutHeader: true,
	}))

	router.GET("/fast", func(ctx *chain.Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Timeout-Seconds") != "5s" {
		t.Errorf("expected X-Timeout-Seconds header to be '5s', got %q", w.Header().Get("X-Timeout-Seconds"))
	}
}

// ============================================================================
// Context Cancellation Tests
// ============================================================================

func Test_New_ContextCancellation_DatabaseSimulation(t *testing.T) {
	router := chain.New()

	transactionRolledBack := false

	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	router.GET("/db", func(ctx *chain.Context) error {
		// Simulate starting a database transaction
		txCtx, txCancel := context.WithCancel(ctx.Request.Context())
		defer func() {
			txCancel()
			transactionRolledBack = true
		}()

		// Simulate long database operation that respects context
		select {
		case <-txCtx.Done():
			// Transaction context cancelled - should roll back
			return txCtx.Err()
		case <-time.After(200 * time.Millisecond):
			// Simulated query completed
			ctx.OK()
			return nil
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/db", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !transactionRolledBack {
		t.Error("database transaction should have been rolled back")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func Test_New_ContextCancellation_HTTPClientSimulation(t *testing.T) {
	router := chain.New()

	var httpRequestCancelled bool

	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	router.GET("/http", func(ctx *chain.Context) error {
		// Simulate an HTTP client request that respects context
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Request.Context().Done():
				httpRequestCancelled = true
				return ctx.Request.Context().Err()
			case <-time.After(10 * time.Millisecond):
				// Keep waiting
			}
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/http", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !httpRequestCancelled {
		t.Error("HTTP request should have been cancelled")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// ============================================================================
// Concurrent Request Tests
// ============================================================================

func Test_New_ConcurrentRequests(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: 100 * time.Millisecond,
	}))

	var completedCount int64
	router.GET("/concurrent", func(ctx *chain.Context) error {
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt64(&completedCount, 1)
		ctx.OK()
		return nil
	})

	const numRequests = 20
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/concurrent", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}
		}()
	}

	wg.Wait()

	if completedCount != numRequests {
		t.Errorf("expected %d completed handlers, got %d", numRequests, completedCount)
	}
}

func Test_New_ConcurrentTimeouts(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	var timedOutCount int64
	router.GET("/timeout", func(ctx *chain.Context) error {
		time.Sleep(100 * time.Millisecond)
		ctx.OK()
		return nil
	})

	const numRequests = 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/timeout", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusServiceUnavailable {
				atomic.AddInt64(&timedOutCount, 1)
			}
		}()
	}

	wg.Wait()

	if timedOutCount != numRequests {
		t.Errorf("expected %d timed out requests, got %d", numRequests, timedOutCount)
	}
}

// ============================================================================
// Middleware Chain Tests
// ============================================================================

func Test_New_WithOtherMiddlewares(t *testing.T) {
	router := chain.New()

	var order []string
	var mu sync.Mutex

	// Logging middleware before timeout
	router.Use(func(ctx *chain.Context, next func() error) error {
		mu.Lock()
		order = append(order, "log-before")
		mu.Unlock()
		err := next()
		mu.Lock()
		order = append(order, "log-after")
		mu.Unlock()
		return err
	})

	// Timeout middleware
	router.Use(New(Config{
		Timeout: 200 * time.Millisecond,
	}))

	router.GET("/test", func(ctx *chain.Context) error {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := []string{"log-before", "handler", "log-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(order), order)
	}
	for i, step := range expected {
		if order[i] != step {
			t.Errorf("step %d: expected %q, got %q", i, step, order[i])
		}
	}
}

func Test_New_PathScoped(t *testing.T) {
	router := chain.New()

	// Timeout only for /api/* routes
	router.Use("/api/*", New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	router.GET("/api/slow", func(ctx *chain.Context) error {
		time.Sleep(100 * time.Millisecond)
		ctx.OK()
		return nil
	})

	publicHandlerCompleted := false
	router.GET("/public/slow", func(ctx *chain.Context) error {
		time.Sleep(50 * time.Millisecond)
		publicHandlerCompleted = true
		ctx.OK()
		return nil
	})

	// Test API route (should timeout)
	req := httptest.NewRequest(http.MethodGet, "/api/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d for /api/slow, got %d", http.StatusServiceUnavailable, w.Code)
	}

	// Test public route (should complete)
	req = httptest.NewRequest(http.MethodGet, "/public/slow", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for /public/slow, got %d", http.StatusOK, w.Code)
	}
	if !publicHandlerCompleted {
		t.Error("Public handler should have completed")
	}
}

func Test_New_HandlerReturnsError(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: 200 * time.Millisecond,
	}))

	handlerCalled := false
	router.GET("/error", func(ctx *chain.Context) error {
		handlerCalled = true
		return context.Canceled
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should have been called")
	}
}

func Test_New_AlreadyWrittenResponse(t *testing.T) {
	router := chain.New()

	router.Use(New(Config{
		Timeout: 200 * time.Millisecond,
	}))

	// Handler writes to response before timeout
	router.GET("/written", func(ctx *chain.Context) error {
		// Write response immediately
		ctx.OK()
		// Now wait (this won't affect the response)
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/written", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Response should have been written successfully
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// ============================================================================
// Error Value Tests
// ============================================================================

func Test_New_ErrRequestTimeout(t *testing.T) {
	router := chain.New()

	var capturedErr error
	router.ErrorHandler = func(ctx *chain.Context, err error) {
		capturedErr = err
	}

	router.Use(New(Config{
		Timeout: 50 * time.Millisecond,
	}))

	router.GET("/timeout", func(ctx *chain.Context) error {
		// Handler cooperates with context cancellation
		for i := 0; i < 20; i++ {
			select {
			case <-ctx.Request.Context().Done():
				return ctx.Request.Context().Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/timeout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedErr != ErrRequestTimeout {
		t.Errorf("expected error %v, got %v", ErrRequestTimeout, capturedErr)
	}
}

// ============================================================================
// Convenience Function Tests
// ============================================================================

func Test_WithTimeout(t *testing.T) {
	router := chain.New()

	router.Use(WithTimeout(200 * time.Millisecond))

	router.GET("/fast", func(ctx *chain.Context) error {
		time.Sleep(50 * time.Millisecond)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected default status code %d, got %d", http.StatusServiceUnavailable, cfg.StatusCode)
	}
	if cfg.IncludeTimeoutHeader != false {
		t.Error("expected IncludeTimeoutHeader to be false by default")
	}
}
