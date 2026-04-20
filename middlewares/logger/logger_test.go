package logger

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nidorx/chain"
)

func performRequestLog(r http.Handler, method string, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLoggerDefault(t *testing.T) {
	router := chain.New()
	router.Use(New())

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerWithConfig(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		Format: FormatDefault,
		Logger: slog.Default(),
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerSkipPaths(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		SkipPaths: []string{"/health"},
	}))

	router.GET("/health", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/health", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerSkipPathPrefixes(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		SkipPathPrefixes: []string{"/static/", "/assets/"},
	}))

	router.GET("/static/file.js", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/static/file.js", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerRequestID(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		RequestIDHeader: "X-Request-ID",
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", map[string]string{"X-Request-ID": "test-request-id"})
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerGenerateRequestID(t *testing.T) {
	// Test that the middleware runs without error
	router := chain.New()
	router.Use(New(Config{
		GenerateRequestID: true,
		Logger:            nil,
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerGetRequestID(t *testing.T) {
	// Test that the middleware runs without error
	router := chain.New()
	router.Use(New())

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerFormatCombined(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		Format: FormatCombined,
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerFormatCustom(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		Format:       FormatCustom,
		CustomFormat: "%{method} %{path} %{status} %{latency}",
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerStatusLevels(t *testing.T) {
	// Test that different status codes are handled correctly
	testCases := []struct {
		status int
		path   string
	}{
		{200, "/ok"},
		{400, "/bad"},
		{404, "/notfound"},
		{500, "/error"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			router := chain.New()
			router.Use(New())

			router.GET(tc.path, func(ctx *chain.Context) {
				ctx.Status(tc.status)
			})

			w := performRequestLog(router, "GET", tc.path, nil)
			if w.Code != tc.status {
				t.Errorf("Expected status %d, got %d", tc.status, w.Code)
			}
		})
	}
}

func TestLoggerLatencyThreshold(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		LatencyThreshold: time.Millisecond,
	}))

	router.GET("/slow", func(ctx *chain.Context) {
		time.Sleep(2 * time.Millisecond)
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/slow", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerStatusLevelFunc(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		StatusLevelFunc: func(status int) slog.Level {
			if status >= 400 {
				return slog.LevelError
			}
			return slog.LevelDebug
		},
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggerWithError(t *testing.T) {
	router := chain.New()
	router.Use(New())

	router.GET("/error", func(ctx *chain.Context) error {
		ctx.BadRequest("error")
		return nil
	})

	w := performRequestLog(router, "GET", "/error", nil)
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestLoggerCustomFormatPlaceholders(t *testing.T) {
	format := "%{method} %{path} %{status} %{latency} %{ip} %{useragent} %{referer} %{host} %{proto} %{reqid} %{err} %{query}"
	result := formatCustom(format, &chain.Context{
		Request: httptest.NewRequest("GET", "/test?foo=bar", nil),
	}, 200, time.Second, "test-id", nil)

	if !strings.Contains(result, "GET") {
		t.Error("Expected method in output")
	}
	if !strings.Contains(result, "/test") {
		t.Error("Expected path in output")
	}
	if !strings.Contains(result, "200") {
		t.Error("Expected status in output")
	}
	if !strings.Contains(result, "test-id") {
		t.Error("Expected request ID in output")
	}
}

func TestLoggerDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Format != FormatDefault {
		t.Errorf("Expected default format to be FormatDefault, got %s", cfg.Format)
	}
	if cfg.RequestIDHeader != "X-Request-ID" {
		t.Errorf("Expected default request ID header to be X-Request-ID, got %s", cfg.RequestIDHeader)
	}
	if !cfg.GenerateRequestID {
		t.Error("Expected GenerateRequestID to be true")
	}
}

func TestLoggerWithNilLogger(t *testing.T) {
	router := chain.New()
	router.Use(New(Config{
		Logger: nil,
	}))

	router.GET("/test", func(ctx *chain.Context) {
		ctx.OK("ok")
	})

	w := performRequestLog(router, "GET", "/test", nil)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
