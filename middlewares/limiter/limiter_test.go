package limiter

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nidorx/chain"
)

func performRequestLimiter(r http.Handler, method string, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLimiterDefault(t *testing.T) {
	router := chain.New()
	router.Use(New())

	router.POST("/test", func(ctx *chain.Context) error {
		ctx.OK("ok")
		return nil
	})

	w := performRequestLimiter(router, "POST", "/test", []byte("test"))
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLimiterSkipMethods(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		MaxBodySize: 100,
		SkipMethods: []string{http.MethodGet},
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLimiter(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLimiterDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxBodySize != 10<<20 {
		t.Errorf("Expected MaxBodySize to be %d, got %d", 10<<20, cfg.MaxBodySize)
	}
	if cfg.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected StatusCode to be %d, got %d", http.StatusRequestEntityTooLarge, cfg.StatusCode)
	}
	if len(cfg.SkipMethods) != 3 {
		t.Errorf("Expected 3 skip methods, got %d", len(cfg.SkipMethods))
	}
}

func TestLimiterHEAD(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		MaxBodySize: 100,
	}))

	router.HEAD("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLimiter(router, "HEAD", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLimiterOPTIONS(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		MaxBodySize: 100,
	}))

	router.OPTIONS("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLimiter(router, "OPTIONS", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLimiterNoBody(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		MaxBodySize: 100,
	}))

	router.POST("/test", func(ctx *chain.Context) error {
		ctx.OK("ok")
		return nil
	})

	w := performRequestLimiter(router, "POST", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLimiterMaxBytesHelper(t *testing.T) {
	router := chain.New()
	router.Use(MaxBytes(100))

	router.POST("/test", func(ctx *chain.Context) error {
		ctx.OK("ok")
		return nil
	})

	w := performRequestLimiter(router, "POST", "/test", []byte("test"))
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
