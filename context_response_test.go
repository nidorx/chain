package chain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Status Method Tests
// ============================================================================

func TestContext_Status_WithNoContent(t *testing.T) {
	router := New()
	router.GET("/status", func(ctx *Context) error {
		ctx.Status(http.StatusAccepted)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
}

func TestContext_Status_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/status", func(ctx *Context) error {
		ctx.Status(http.StatusOK, "OK message")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "OK message" {
		t.Errorf("expected body 'OK message', got '%s'", w.Body.String())
	}
}

func TestContext_Status_WithBytesContent(t *testing.T) {
	router := New()
	router.GET("/status", func(ctx *Context) error {
		ctx.Status(http.StatusCreated, []byte("created"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	if w.Body.String() != "created" {
		t.Errorf("expected body 'created', got '%s'", w.Body.String())
	}
}

func TestContext_Status_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/status", func(ctx *Context) error {
		ctx.Status(http.StatusOK, map[string]string{"message": "hello"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
	expected := `{"message":"hello"}`
	if w.Body.String() != expected {
		t.Errorf("expected body '%s', got '%s'", expected, w.Body.String())
	}
}

// ============================================================================
// OK Method Tests
// ============================================================================

func TestContext_OK_NoContent(t *testing.T) {
	router := New()
	router.GET("/ok", func(ctx *Context) error {
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestContext_OK_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/ok", func(ctx *Context) error {
		ctx.OK("success")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("expected body 'success', got '%s'", w.Body.String())
	}
}

func TestContext_OK_WithBytesContent(t *testing.T) {
	router := New()
	router.GET("/ok", func(ctx *Context) error {
		ctx.OK([]byte("ok bytes"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "ok bytes" {
		t.Errorf("expected body 'ok bytes', got '%s'", w.Body.String())
	}
}

func TestContext_OK_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/ok", func(ctx *Context) error {
		ctx.OK(map[string]string{"status": "ok"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
	expected := `{"status":"ok"}`
	if w.Body.String() != expected {
		t.Errorf("expected body '%s', got '%s'", expected, w.Body.String())
	}
}

// ============================================================================
// Created Method Tests
// ============================================================================

func TestContext_Created_NoContent(t *testing.T) {
	router := New()
	router.GET("/created", func(ctx *Context) error {
		ctx.Created()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/created", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestContext_Created_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/created", func(ctx *Context) error {
		ctx.Created("resource created")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/created", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	if w.Body.String() != "resource created" {
		t.Errorf("expected body 'resource created', got '%s'", w.Body.String())
	}
}

func TestContext_Created_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/created", func(ctx *Context) error {
		ctx.Created(map[string]string{"id": "123", "name": "test"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/created", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
	// JSON may have different key ordering, so we check for both possibilities
	body := w.Body.String()
	if body != `{"id":"123","name":"test"}` && body != `{"name":"test","id":"123"}` {
		t.Errorf("expected JSON body, got '%s'", body)
	}
}

// ============================================================================
// NoContent Method Tests
// ============================================================================

func TestContext_NoContent(t *testing.T) {
	router := New()
	router.GET("/no-content", func(ctx *Context) error {
		ctx.NoContent()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/no-content", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

// ============================================================================
// Error Method Tests
// ============================================================================

func TestContext_Error_Basic(t *testing.T) {
	router := New()
	router.GET("/error", func(ctx *Context) error {
		ctx.Error("Bad Request", http.StatusBadRequest)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if w.Body.String() != "Bad Request\n" {
		t.Errorf("expected body 'Bad Request\\n', got '%s'", w.Body.String())
	}
	if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestContext_Error_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/error", func(ctx *Context) error {
		ctx.Error("ignored", http.StatusBadRequest, "custom error")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if w.Body.String() != "custom error\n" {
		t.Errorf("expected body 'custom error\\n', got '%s'", w.Body.String())
	}
}

func TestContext_Error_WithBytesContent(t *testing.T) {
	router := New()
	router.GET("/error", func(ctx *Context) error {
		ctx.Error("ignored", http.StatusBadRequest, []byte("bytes error"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if w.Body.String() != "bytes error\n" {
		t.Errorf("expected body 'bytes error\\n', got '%s'", w.Body.String())
	}
}

func TestContext_Error_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/error", func(ctx *Context) error {
		ctx.Error("ignored", http.StatusBadRequest, map[string]string{"error": "validation failed"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
	expected := `{"error":"validation failed"}`
	if w.Body.String() != expected {
		t.Errorf("expected body '%s', got '%s'", expected, w.Body.String())
	}
}

// ============================================================================
// BadRequest Method Tests
// ============================================================================

func TestContext_BadRequest_NoContent(t *testing.T) {
	router := New()
	router.GET("/bad", func(ctx *Context) error {
		ctx.BadRequest()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContext_BadRequest_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/bad", func(ctx *Context) error {
		ctx.BadRequest("invalid input")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if w.Body.String() != "invalid input\n" {
		t.Errorf("expected body 'invalid input\\n', got '%s'", w.Body.String())
	}
}

func TestContext_BadRequest_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/bad", func(ctx *Context) error {
		ctx.BadRequest(map[string]string{"error": "invalid json"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// Unauthorized Method Tests
// ============================================================================

func TestContext_Unauthorized_NoContent(t *testing.T) {
	router := New()
	router.GET("/unauthorized", func(ctx *Context) error {
		ctx.Unauthorized()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/unauthorized", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestContext_Unauthorized_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/unauthorized", func(ctx *Context) error {
		ctx.Unauthorized("login required")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/unauthorized", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if w.Body.String() != "login required\n" {
		t.Errorf("expected body 'login required\\n', got '%s'", w.Body.String())
	}
}

func TestContext_Unauthorized_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/unauthorized", func(ctx *Context) error {
		ctx.Unauthorized(map[string]string{"error": "invalid token"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/unauthorized", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// Forbidden Method Tests
// ============================================================================

func TestContext_Forbidden_NoContent(t *testing.T) {
	router := New()
	router.GET("/forbidden", func(ctx *Context) error {
		ctx.Forbidden()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestContext_Forbidden_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/forbidden", func(ctx *Context) error {
		ctx.Forbidden("access denied")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if w.Body.String() != "access denied\n" {
		t.Errorf("expected body 'access denied\\n', got '%s'", w.Body.String())
	}
}

func TestContext_Forbidden_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/forbidden", func(ctx *Context) error {
		ctx.Forbidden(map[string]string{"error": "insufficient permissions"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// NotFound Method Tests
// ============================================================================

func TestContext_NotFound_NoContent(t *testing.T) {
	router := New()
	router.GET("/notfound", func(ctx *Context) error {
		ctx.NotFound()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestContext_NotFound_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/notfound", func(ctx *Context) error {
		ctx.NotFound("resource missing")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	if w.Body.String() != "resource missing\n" {
		t.Errorf("expected body 'resource missing\\n', got '%s'", w.Body.String())
	}
}

func TestContext_NotFound_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/notfound", func(ctx *Context) error {
		ctx.NotFound(map[string]string{"error": "not found"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// TooManyRequests Method Tests
// ============================================================================

func TestContext_TooManyRequests_NoContent(t *testing.T) {
	router := New()
	router.GET("/ratelimit", func(ctx *Context) error {
		ctx.TooManyRequests()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ratelimit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestContext_TooManyRequests_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/ratelimit", func(ctx *Context) error {
		ctx.TooManyRequests("rate limit exceeded")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ratelimit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
	if w.Body.String() != "rate limit exceeded\n" {
		t.Errorf("expected body 'rate limit exceeded\\n', got '%s'", w.Body.String())
	}
}

func TestContext_TooManyRequests_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/ratelimit", func(ctx *Context) error {
		ctx.TooManyRequests(map[string]any{"retry_after": 60})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ratelimit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// InternalServerError Method Tests
// ============================================================================

func TestContext_InternalServerError_NoContent(t *testing.T) {
	router := New()
	router.GET("/error500", func(ctx *Context) error {
		ctx.InternalServerError()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error500", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestContext_InternalServerError_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/error500", func(ctx *Context) error {
		ctx.InternalServerError("server error")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error500", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if w.Body.String() != "server error\n" {
		t.Errorf("expected body 'server error\\n', got '%s'", w.Body.String())
	}
}

func TestContext_InternalServerError_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/error500", func(ctx *Context) error {
		ctx.InternalServerError(map[string]string{"error": "internal error"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error500", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// NotImplemented Method Tests
// ============================================================================

func TestContext_NotImplemented_NoContent(t *testing.T) {
	router := New()
	router.GET("/notimpl", func(ctx *Context) error {
		ctx.NotImplemented()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notimpl", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status %d, got %d", http.StatusNotImplemented, w.Code)
	}
}

func TestContext_NotImplemented_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/notimpl", func(ctx *Context) error {
		ctx.NotImplemented("feature not available")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notimpl", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status %d, got %d", http.StatusNotImplemented, w.Code)
	}
	if w.Body.String() != "feature not available\n" {
		t.Errorf("expected body 'feature not available\\n', got '%s'", w.Body.String())
	}
}

func TestContext_NotImplemented_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/notimpl", func(ctx *Context) error {
		ctx.NotImplemented(map[string]string{"error": "not implemented"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notimpl", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status %d, got %d", http.StatusNotImplemented, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// ServiceUnavailable Method Tests
// ============================================================================

func TestContext_ServiceUnavailable_NoContent(t *testing.T) {
	router := New()
	router.GET("/unavailable", func(ctx *Context) error {
		ctx.ServiceUnavailable()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/unavailable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestContext_ServiceUnavailable_WithStringContent(t *testing.T) {
	router := New()
	router.GET("/unavailable", func(ctx *Context) error {
		ctx.ServiceUnavailable("maintenance")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/unavailable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
	if w.Body.String() != "maintenance\n" {
		t.Errorf("expected body 'maintenance\\n', got '%s'", w.Body.String())
	}
}

func TestContext_ServiceUnavailable_WithJSONContent(t *testing.T) {
	router := New()
	router.GET("/unavailable", func(ctx *Context) error {
		ctx.ServiceUnavailable(map[string]string{"error": "service unavailable"})
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/unavailable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

// ============================================================================
// Comprehensive Response Tests
// ============================================================================

func TestContext_ResponseWriting_AllStatusCodes(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"OK", "/ok", http.StatusOK},
		{"Created", "/created", http.StatusCreated},
		{"NoContent", "/no-content", http.StatusNoContent},
		{"BadRequest", "/bad-request", http.StatusBadRequest},
		{"Unauthorized", "/unauthorized", http.StatusUnauthorized},
		{"Forbidden", "/forbidden", http.StatusForbidden},
		{"NotFound", "/not-found", http.StatusNotFound},
		{"TooManyRequests", "/too-many-requests", http.StatusTooManyRequests},
		{"InternalServerError", "/internal-error", http.StatusInternalServerError},
		{"NotImplemented", "/not-implemented", http.StatusNotImplemented},
		{"ServiceUnavailable", "/service-unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New()
			router.GET(tt.path, func(ctx *Context) error {
				switch tt.status {
				case http.StatusOK:
					ctx.OK()
				case http.StatusCreated:
					ctx.Created()
				case http.StatusNoContent:
					ctx.NoContent()
				case http.StatusBadRequest:
					ctx.BadRequest()
				case http.StatusUnauthorized:
					ctx.Unauthorized()
				case http.StatusForbidden:
					ctx.Forbidden()
				case http.StatusNotFound:
					ctx.NotFound()
				case http.StatusTooManyRequests:
					ctx.TooManyRequests()
				case http.StatusInternalServerError:
					ctx.InternalServerError()
				case http.StatusNotImplemented:
					ctx.NotImplemented()
				case http.StatusServiceUnavailable:
					ctx.ServiceUnavailable()
				}
				return nil
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func TestContext_ResponseWriting_WithJSONContent(t *testing.T) {
	router := New()

	router.GET("/ok-json", func(ctx *Context) error {
		ctx.OK(map[string]string{"message": "ok"})
		return nil
	})

	router.GET("/created-json", func(ctx *Context) error {
		ctx.Created(map[string]string{"id": "1"})
		return nil
	})

	router.GET("/bad-json", func(ctx *Context) error {
		ctx.BadRequest(map[string]string{"error": "bad"})
		return nil
	})

	tests := []struct {
		path        string
		status      int
		contentType string
	}{
		{"/ok-json", http.StatusOK, "application/json"},
		{"/created-json", http.StatusCreated, "application/json"},
		{"/bad-json", http.StatusBadRequest, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}
			if w.Header().Get("Content-Type") != tt.contentType {
				t.Errorf("expected Content-Type '%s', got '%s'", tt.contentType, w.Header().Get("Content-Type"))
			}
		})
	}
}
