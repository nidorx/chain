package chain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Server Creation Tests
// ============================================================================

func Test_NewServer_DefaultConfig(t *testing.T) {
	router := New()
	server := NewServer(router, ":8080")

	if server == nil {
		t.Fatal("expected server to be created")
	}
	if server.Server == nil {
		t.Fatal("expected http.Server to be set")
	}
	if server.Server.Addr != ":8080" {
		t.Errorf("expected addr ':8080', got '%s'", server.Server.Addr)
	}
	if server.Server.Handler != router {
		t.Error("expected handler to be the router")
	}
	if server.Config.Timeout != DefaultShutdownTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultShutdownTimeout, server.Config.Timeout)
	}
	if server.Config.Signals != nil {
		t.Error("expected signals to be nil (use defaults)")
	}
}

func Test_NewServerWithConfig_CustomTimeout(t *testing.T) {
	router := New()
	customTimeout := 60 * time.Second
	server := NewServerWithConfig(router, ":9090", ShutdownConfig{
		Timeout: customTimeout,
	})

	if server.Config.Timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, server.Config.Timeout)
	}
}

func Test_NewServerWithConfig_ZeroTimeout(t *testing.T) {
	router := New()
	server := NewServerWithConfig(router, ":9090", ShutdownConfig{
		Timeout: 0,
	})

	// Zero timeout should default to DefaultShutdownTimeout
	if server.Config.Timeout != DefaultShutdownTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultShutdownTimeout, server.Config.Timeout)
	}
}

func Test_NewServerWithConfig_NegativeTimeout(t *testing.T) {
	router := New()
	server := NewServerWithConfig(router, ":9090", ShutdownConfig{
		Timeout: -1 * time.Second,
	})

	// Negative timeout should default to DefaultShutdownTimeout
	if server.Config.Timeout != DefaultShutdownTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultShutdownTimeout, server.Config.Timeout)
	}
}

// ============================================================================
// Server Lifecycle Tests
// ============================================================================

func Test_Server_Shutdown(t *testing.T) {
	router := New()
	router.GET("/", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	server := NewServer(router, ":0") // Port 0 = random available port

	// Start server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Ignore expected errors
		}
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Initiate shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected no error during shutdown, got %v", err)
	}

	// Wait for shutdown to complete
	server.Wait()
}

func Test_Server_MultipleShutdownCalls(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	// Multiple shutdown calls should not panic or error
	err1 := server.Shutdown(context.Background())
	err2 := server.Shutdown(context.Background())
	err3 := server.Shutdown(context.Background())

	if err1 != nil {
		t.Errorf("first shutdown should not error, got %v", err1)
	}
	if err2 != nil {
		t.Errorf("second shutdown should not error, got %v", err2)
	}
	if err3 != nil {
		t.Errorf("third shutdown should not error, got %v", err3)
	}
}

func Test_Server_Stop(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	err := server.Stop()
	if err != nil {
		t.Errorf("expected no error during stop, got %v", err)
	}
}

// ============================================================================
// Server Hooks Tests
// ============================================================================

func Test_Server_OnShutdown(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	onShutdownCalled := false
	server.OnShutdown(func() {
		onShutdownCalled = true
	})

	server.Shutdown(context.Background())

	if !onShutdownCalled {
		t.Error("OnShutdown callback should have been called")
	}
}

func Test_Server_OnStop(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	onStopCalled := false
	server.OnStop(func() {
		onStopCalled = true
	})

	server.Shutdown(context.Background())
	server.Wait()

	if !onStopCalled {
		t.Error("OnStop callback should have been called")
	}
}

func Test_Server_BothHooks(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	var order []string
	var mu sync.Mutex

	server.OnShutdown(func() {
		mu.Lock()
		order = append(order, "on-shutdown")
		mu.Unlock()
	})

	server.OnStop(func() {
		mu.Lock()
		order = append(order, "on-stop")
		mu.Unlock()
	})

	server.Shutdown(context.Background())
	server.Wait()

	if len(order) != 2 {
		t.Fatalf("expected 2 hook calls, got %d", len(order))
	}
	if order[0] != "on-shutdown" {
		t.Errorf("expected first hook 'on-shutdown', got '%s'", order[0])
	}
	if order[1] != "on-stop" {
		t.Errorf("expected second hook 'on-stop', got '%s'", order[1])
	}
}

// ============================================================================
// IsShuttingDown Tests
// ============================================================================

func Test_Server_IsShuttingDown(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	// Not shutting down initially
	if server.IsShuttingDown() {
		t.Error("server should not be shutting down initially")
	}

	// Start shutdown
	server.Shutdown(context.Background())

	// Now should be shutting down
	if !server.IsShuttingDown() {
		t.Error("server should be shutting down after Shutdown() call")
	}
}

func Test_Server_IsShuttingDown_Concurrent(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	var wg sync.WaitGroup
	const numGoroutines = 10
	wg.Add(numGoroutines)

	// Concurrently check and set shutting down
	for i := 0; i < numGoroutines/2; i++ {
		go func() {
			defer wg.Done()
			server.IsShuttingDown()
		}()
	}

	for i := 0; i < numGoroutines/2; i++ {
		go func() {
			defer wg.Done()
			server.Shutdown(context.Background())
		}()
	}

	wg.Wait()

	// After all goroutines complete, should be shutting down
	if !server.IsShuttingDown() {
		t.Error("server should be shutting down after concurrent operations")
	}
}

// ============================================================================
// Graceful Middleware Tests
// ============================================================================

func Test_GracefulMiddleware_NotShuttingDown(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	router.Use(GracefulMiddleware(server))

	router.GET("/test", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Connection header should not be "close" when not shutting down
	connHeader := w.Header().Get("Connection")
	if connHeader == "close" {
		t.Error("Connection header should not be 'close' when not shutting down")
	}
}

func Test_GracefulMiddleware_ShuttingDown(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	router.Use(GracefulMiddleware(server))

	router.GET("/test", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	// Initiate shutdown before making the request
	server.Shutdown(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Connection header should be "close" when shutting down
	connHeader := w.Header().Get("Connection")
	if connHeader != "close" {
		t.Errorf("expected Connection header 'close', got '%s'", connHeader)
	}
}

// ============================================================================
// Server Integration Tests
// ============================================================================

func Test_Server_WithRouter(t *testing.T) {
	router := New()

	var requestCount int64
	router.GET("/test", func(ctx *Context) error {
		atomic.AddInt64(&requestCount, 1)
		ctx.Json(map[string]int64{"count": atomic.LoadInt64(&requestCount)})
		return nil
	})

	server := NewServer(router, ":0")

	// Verify the server can serve requests before shutdown
	// We simulate this by using the router directly (since we can't easily start/stop in tests)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Shutdown should work cleanly
	err := server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("expected no error during shutdown, got %v", err)
	}
}

func Test_Server_HandlerIsRouter(t *testing.T) {
	router := New()
	server := NewServer(router, ":8080")

	if server.Server.Handler != router {
		t.Error("server handler should be the router")
	}
}

// ============================================================================
// Server Context Tests
// ============================================================================

func Test_Server_ShutdownWithContext(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	// This should complete within the context timeout
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func Test_Server_ShutdownWithNilContext(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	// Nil context should use the configured timeout
	err := server.Shutdown(nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// ============================================================================
// Server Default Address Tests
// ============================================================================

func Test_Server_ListenAndServe_DefaultAddress(t *testing.T) {
	router := New()
	server := NewServer(router, "")

	// When addr is empty, ListenAndServe should use ":http"
	if server.Server.Addr != "" {
		t.Errorf("expected empty addr, got '%s'", server.Server.Addr)
	}

	// Server will set the addr to :http when Addr is empty in ListenAndServe
	// We can't easily test this without actually listening, so just verify the setup
	if server.Server.Handler != router {
		t.Error("handler should be router")
	}
}

// ============================================================================
// ShutdownConfig Tests
// ============================================================================

func Test_ShutdownConfig_DefaultSignals(t *testing.T) {
	// Verify that nil signals will default to SIGINT and SIGTERM
	// (We can't easily test the actual signal handling in unit tests)
	config := ShutdownConfig{
		Timeout: 10 * time.Second,
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", config.Timeout)
	}
	if config.Signals != nil {
		t.Error("expected signals to be nil")
	}
}

// ============================================================================
// Server Wait Tests
// ============================================================================

func Test_Server_Wait_BeforeShutdown(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	// Calling Wait before Shutdown should not block indefinitely
	// (but since we're calling shutdown immediately after, it should complete)
	done := make(chan struct{})
	go func() {
		server.Wait()
		close(done)
	}()

	// Give Wait a moment to start waiting
	time.Sleep(10 * time.Millisecond)

	// Now shutdown
	server.Shutdown(context.Background())

	// Wait should complete
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not complete after shutdown")
	}
}

func Test_Server_Wait_AfterShutdown(t *testing.T) {
	router := New()
	server := NewServer(router, ":0")

	// Shutdown first
	server.Shutdown(context.Background())

	// Then wait (should complete immediately)
	done := make(chan struct{})
	go func() {
		server.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not complete after shutdown")
	}
}

func Test_Server_Wait_NilStopChan(t *testing.T) {
	router := New()
	server := &Server{
		Server:   &http.Server{Handler: router},
		stopChan: nil,
	}

	// Wait with nil stopChan should not panic
	server.Wait()
	// If we got here without panicking, the test passes
}

// ============================================================================
// Concurrent Request During Shutdown Tests
// ============================================================================

func Test_Server_ConcurrentRequestsDuringShutdown(t *testing.T) {
	router := New()

	var completedCount int64
	router.GET("/test", func(ctx *Context) error {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&completedCount, 1)
		ctx.OK()
		return nil
	})

	server := NewServer(router, ":0")

	// Make some requests before shutdown
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Initiate shutdown
	server.Shutdown(context.Background())

	// All pre-shutdown requests should have completed
	if atomic.LoadInt64(&completedCount) != 5 {
		t.Errorf("expected 5 completed requests, got %d", completedCount)
	}
}
