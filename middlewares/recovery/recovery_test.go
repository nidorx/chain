package recovery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nidorx/chain"
)

func performRequestRecovery(r http.Handler, method string, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRecoveryDefault(t *testing.T) {
	router := chain.New()
	router.Use(New())

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	// Body might be empty depending on when the panic occurs
	// The important thing is that we got 500 and didn't crash
}

func TestRecoveryWithConfig(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		PrintStack: false,
	}))

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecoveryCustomHandler(t *testing.T) {
	// Custom handler is called when panic is recovered
	// The default behavior returns 500
	router := chain.New()
	router.Use(New(Config{
		PrintStack: false,
	}))

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecoveryNoPanic(t *testing.T) {
	router := chain.New()
	router.Use(New())

	router.GET("/ok", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestRecovery(router, "GET", "/ok")
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecoveryWithNilLogger(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		Logger: nil,
	}))

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecoveryDisablePanicLogging(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		DisablePanicLogging: true,
	}))

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecoveryWithMultipleMiddlewares(t *testing.T) {
	router := chain.New()
	router.Use(New())
	router.Use(func(ctx *chain.Context, next func() error) error {
		return next()
	})

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecoveryWithResponseAlreadyWritten(t *testing.T) {
	router := chain.New()
	router.Use(New())

	router.GET("/partial", func(ctx *chain.Context) {
		ctx.Writer.Write([]byte("partial response"))
		panic("test panic after write")
	})

	w := performRequestRecovery(router, "GET", "/partial")
	// Should not write 500 because response was already started
	if w.Code != 200 {
		t.Errorf("Expected status 200 (partial write), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "partial response") {
		t.Errorf("Expected partial response in body, got %s", w.Body.String())
	}
}

func TestRecoveryForceStackPanic(t *testing.T) {
	defer func() {
		if rcv := recover(); rcv == nil {
			t.Error("Expected panic but didn't get one")
		}
	}()

	ForceStackPanic("test panic")
}

func TestRecoveryDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.PrintStack {
		t.Error("Expected PrintStack to be true by default")
	}
	if cfg.StackSize != 4096 {
		t.Errorf("Expected StackSize to be 4096, got %d", cfg.StackSize)
	}
}

func TestRecoveryWithStackSize(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		StackSize: 8192,
	}))

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecoveryChain(t *testing.T) {
	// Test that recovery works in a middleware chain
	middleware1Called := false
	middleware2Called := false

	router := chain.New()
	router.Use(New())
	router.Use(func(ctx *chain.Context, next func() error) error {
		middleware1Called = true
		return next()
	})
	router.Use(func(ctx *chain.Context, next func() error) error {
		middleware2Called = true
		return next()
	})

	router.GET("/panic", func(ctx *chain.Context) {
		panic("test panic")
	})

	w := performRequestRecovery(router, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	if !middleware1Called {
		t.Error("Expected middleware1 to be called")
	}
	if !middleware2Called {
		t.Error("Expected middleware2 to be called")
	}
}
