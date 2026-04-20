package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nidorx/chain"
)

func performRequest(r http.Handler, method string, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCorsDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if len(config.AllowMethods) != 7 {
		t.Errorf("Expected 7 default methods, got %d", len(config.AllowMethods))
	}
	if len(config.AllowHeaders) != 3 {
		t.Errorf("Expected 3 default headers, got %d", len(config.AllowHeaders))
	}
	if config.MaxAge != 12*time.Hour {
		t.Errorf("Expected 12h max age, got %v", config.MaxAge)
	}
}

func TestCorsDefault(t *testing.T) {
	router := chain.New()
	router.Use(Default())

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://example.com"})

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCorsNew(t *testing.T) {
	config := Config{
		AllowOrigins: []string{"http://example.com"},
	}
	mw := New(config)
	if mw == nil {
		t.Error("Expected middleware to be created")
	}
}

func TestCorsConfigValidation(t *testing.T) {
	config := Config{
		AllowAllOrigins: true,
		AllowOrigins:    []string{"http://example.com"},
	}
	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for conflicting settings")
	}
}

func TestCorsNoOriginHeader(t *testing.T) {
	router := chain.New()
	router.Use(Default())

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequest(router, "GET", "/test", nil)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCorsAllowedOrigins(t *testing.T) {
	config := Config{
		AllowOrigins: []string{"http://allowed.com"},
	}
	router := chain.New()
	router.Use(New(config))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	// Allowed origin
	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://allowed.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Disallowed origin should return 403
	w = performRequest(router, "GET", "/test", map[string]string{"Origin": "http://disallowed.com"})
	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCorsPreflight(t *testing.T) {
	config := Config{
		AllowOrigins:     []string{"http://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"X-Custom-Header"},
		AllowCredentials: true,
		MaxAge:           time.Hour,
	}
	router := chain.New()
	router.Use(New(config))

	router.OPTIONS("/test", func(ctx *chain.Context) {
		ctx.NoContent()
	})

	w := performRequest(router, "OPTIONS", "/test", map[string]string{
		"Origin":                        "http://example.com",
		"Access-Control-Request-Method": "POST",
	})

	if w.Code != 204 {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestCorsWildcardOrigins(t *testing.T) {
	config := Config{
		AllowOrigins:  []string{"http://*.example.com"},
		AllowWildcard: true,
	}
	router := chain.New()
	router.Use(New(config))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	// Matching wildcard
	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://sub.example.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Not matching
	w = performRequest(router, "GET", "/test", map[string]string{"Origin": "http://other.com"})
	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCorsAllowAllOrigins(t *testing.T) {
	config := Config{
		AllowAllOrigins: true,
	}
	router := chain.New()
	router.Use(New(config))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://any.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCorsOriginFunc(t *testing.T) {
	config := Config{
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://allowed.com"
		},
	}
	router := chain.New()
	router.Use(New(config))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	// Allowed
	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://allowed.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Not allowed
	w = performRequest(router, "GET", "/test", map[string]string{"Origin": "http://disallowed.com"})
	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCorsOriginWithContextFunc(t *testing.T) {
	config := Config{
		AllowOriginWithContextFunc: func(c *chain.Context, origin string) bool {
			return origin == "http://allowed.com"
		},
	}
	router := chain.New()
	router.Use(New(config))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://allowed.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCorsRegexOrigin(t *testing.T) {
	config := Config{
		AllowOrigins: []string{"/^http://.*\\.example\\.com$/"},
	}
	router := chain.New()
	router.Use(New(config))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequest(router, "GET", "/test", map[string]string{"Origin": "http://sub.example.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCorsAddMethods(t *testing.T) {
	config := DefaultConfig()
	config.AddAllowMethods("CONNECT", "TRACE")
	if len(config.AllowMethods) != 9 {
		t.Errorf("Expected 9 methods after adding, got %d", len(config.AllowMethods))
	}
}

func TestCorsAddHeaders(t *testing.T) {
	config := DefaultConfig()
	config.AddAllowHeaders("Authorization", "X-Requested-With")
	if len(config.AllowHeaders) != 5 {
		t.Errorf("Expected 5 headers after adding, got %d", len(config.AllowHeaders))
	}
}

func TestCorsAddExposeHeaders(t *testing.T) {
	config := DefaultConfig()
	config.AddExposeHeaders("X-Request-Id")
	if len(config.ExposeHeaders) != 1 {
		t.Errorf("Expected 1 expose header after adding, got %d", len(config.ExposeHeaders))
	}
}

func TestCorsCustomOptionsStatusCode(t *testing.T) {
	config := Config{
		AllowAllOrigins:           true,
		OptionsResponseStatusCode: http.StatusOK,
	}
	router := chain.New()
	router.Use(New(config))

	router.OPTIONS("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequest(router, "OPTIONS", "/test", map[string]string{"Origin": "http://example.com"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
