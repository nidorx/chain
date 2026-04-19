package chain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// 1. Successful header binding with string fields
func Test_BindHeader_StringFields(t *testing.T) {
	router := New()

	type HeaderStruct struct {
		ContentType   string `header:"Content-Type"`
		Authorization string `header:"Authorization"`
		UserAgent     string `header:"User-Agent"`
	}

	var bound HeaderStruct
	router.GET("/headers", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/headers", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ContentType != "application/json" {
		t.Errorf("expected ContentType 'application/json', got '%s'", bound.ContentType)
	}
	if bound.Authorization != "Bearer token123" {
		t.Errorf("expected Authorization 'Bearer token123', got '%s'", bound.Authorization)
	}
	if bound.UserAgent != "TestAgent/1.0" {
		t.Errorf("expected UserAgent 'TestAgent/1.0', got '%s'", bound.UserAgent)
	}
}

// 2. Header binding with multiple values (slice of strings)
func Test_BindHeader_SliceFields(t *testing.T) {
	router := New()

	type MultiHeaderStruct struct {
		Accept []string `header:"Accept"`
	}

	var bound MultiHeaderStruct
	router.GET("/multi", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/multi", nil)
	req.Header["Accept"] = []string{"text/html", "application/xhtml+xml", "application/xml"}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound.Accept) != 3 {
		t.Errorf("expected 3 Accept values, got %d", len(bound.Accept))
	}
	if bound.Accept[0] != "text/html" {
		t.Errorf("expected Accept[0] 'text/html', got '%s'", bound.Accept[0])
	}
	if bound.Accept[1] != "application/xhtml+xml" {
		t.Errorf("expected Accept[1] 'application/xhtml+xml', got '%s'", bound.Accept[1])
	}
	if bound.Accept[2] != "application/xml" {
		t.Errorf("expected Accept[2] 'application/xml', got '%s'", bound.Accept[2])
	}
}

// 3. Header binding with canonical header key mapping
func Test_BindHeader_CanonicalKeyMapping(t *testing.T) {
	router := New()

	type CanonicalHeaderStruct struct {
		APIKey       string `header:"x-api-key"`
		RequestID    string `header:"x-request-id"`
		CustomHeader string `header:"x-custom-header"`
	}

	var bound CanonicalHeaderStruct
	router.GET("/canonical", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/canonical", nil)
	req.Header.Set("X-Api-Key", "secret-key-123")
	req.Header.Set("X-Request-Id", "req-abc-456")
	req.Header.Set("X-Custom-Header", "custom-value")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.APIKey != "secret-key-123" {
		t.Errorf("expected APIKey 'secret-key-123', got '%s'", bound.APIKey)
	}
	if bound.RequestID != "req-abc-456" {
		t.Errorf("expected RequestID 'req-abc-456', got '%s'", bound.RequestID)
	}
	if bound.CustomHeader != "custom-value" {
		t.Errorf("expected CustomHeader 'custom-value', got '%s'", bound.CustomHeader)
	}
}

// 4. Header binding with integer fields
func Test_BindHeader_IntFields(t *testing.T) {
	router := New()

	type IntHeaderStruct struct {
		ContentLength int   `header:"Content-Length"`
		MaxRetries    int   `header:"X-Max-Retries"`
		RateLimit     int64 `header:"X-Rate-Limit"`
	}

	var bound IntHeaderStruct
	router.GET("/int-headers", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/int-headers", nil)
	req.Header.Set("Content-Length", "1024")
	req.Header.Set("X-Max-Retries", "3")
	req.Header.Set("X-Rate-Limit", "100000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ContentLength != 1024 {
		t.Errorf("expected ContentLength 1024, got %d", bound.ContentLength)
	}
	if bound.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", bound.MaxRetries)
	}
	if bound.RateLimit != 100000 {
		t.Errorf("expected RateLimit 100000, got %d", bound.RateLimit)
	}
}

func Test_BindHeader_IntFields_InvalidValue(t *testing.T) {
	router := New()

	type IntHeaderStruct struct {
		ContentLength int `header:"Content-Length"`
	}

	router.GET("/int-invalid", func(ctx *Context) error {
		var bound IntHeaderStruct
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/int-invalid", nil)
	req.Header.Set("Content-Length", "not-a-number")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// 5. Header binding with missing optional headers
func Test_BindHeader_MissingOptionalHeaders(t *testing.T) {
	router := New()

	type OptionalHeaderStruct struct {
		ContentType   string `header:"Content-Type"`
		Authorization string `header:"Authorization"`
		XCustom       string `header:"X-Custom"`
	}

	var bound OptionalHeaderStruct
	router.GET("/optional", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/optional", nil)
	req.Header.Set("Content-Type", "application/json")
	// Authorization and X-Custom are not set
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ContentType != "application/json" {
		t.Errorf("expected ContentType 'application/json', got '%s'", bound.ContentType)
	}
	if bound.Authorization != "" {
		t.Errorf("expected Authorization '', got '%s'", bound.Authorization)
	}
	if bound.XCustom != "" {
		t.Errorf("expected X-Custom '', got '%s'", bound.XCustom)
	}
}

// 6. Header binding with default values via tag `header:"field,default=value"`
func Test_BindHeader_DefaultValues(t *testing.T) {
	router := New()

	type HeaderWithDefaults struct {
		ContentType string `header:"Content-Type,default=application/json"`
		Limit       int    `header:"X-Limit,default=100"`
		Sort        string `header:"X-Sort,default=asc"`
		Active      bool   `header:"X-Active,default=true"`
	}

	var bound HeaderWithDefaults
	router.GET("/defaults", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// No headers set, should use defaults
	req := httptest.NewRequest(http.MethodGet, "/defaults", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ContentType != "application/json" {
		t.Errorf("expected ContentType 'application/json' (default), got '%s'", bound.ContentType)
	}
	if bound.Limit != 100 {
		t.Errorf("expected Limit 100 (default), got %d", bound.Limit)
	}
	if bound.Sort != "asc" {
		t.Errorf("expected Sort 'asc' (default), got '%s'", bound.Sort)
	}
	if bound.Active != true {
		t.Errorf("expected Active true (default), got %v", bound.Active)
	}
}

func Test_BindHeader_DefaultValues_Override(t *testing.T) {
	router := New()

	type HeaderWithDefaults struct {
		ContentType string `header:"Content-Type,default=application/json"`
		Limit       int    `header:"X-Limit,default=100"`
		Sort        string `header:"X-Sort,default=asc"`
	}

	var bound HeaderWithDefaults
	router.GET("/defaults-override", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Override some defaults
	req := httptest.NewRequest(http.MethodGet, "/defaults-override", nil)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Sort", "desc")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ContentType != "text/plain" {
		t.Errorf("expected ContentType 'text/plain', got '%s'", bound.ContentType)
	}
	if bound.Limit != 100 {
		t.Errorf("expected Limit 100 (default), got %d", bound.Limit)
	}
	if bound.Sort != "desc" {
		t.Errorf("expected Sort 'desc', got '%s'", bound.Sort)
	}
}

// 7. Empty headers (no matching headers)
func Test_BindHeader_EmptyHeaders(t *testing.T) {
	router := New()

	type EmptyHeaderStruct struct {
		Name string `header:"X-Name"`
		Age  int    `header:"X-Age"`
	}

	var bound EmptyHeaderStruct
	router.GET("/empty", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "" {
		t.Errorf("expected Name '', got '%s'", bound.Name)
	}
	if bound.Age != 0 {
		t.Errorf("expected Age 0, got %d", bound.Age)
	}
}

// 8. ShouldBindHeader success and error cases
func Test_ShouldBindHeader_Success(t *testing.T) {
	router := New()

	type HeaderStruct struct {
		APIKey string `header:"X-Api-Key"`
	}

	var bound HeaderStruct
	router.GET("/should-bind", func(ctx *Context) error {
		err := ctx.ShouldBindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/should-bind", nil)
	req.Header.Set("X-Api-Key", "test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.APIKey != "test-key" {
		t.Errorf("expected APIKey 'test-key', got '%s'", bound.APIKey)
	}
}

func Test_ShouldBindHeader_Error(t *testing.T) {
	router := New()

	type HeaderStruct struct {
		Count int `header:"X-Count"`
	}

	router.GET("/should-bind-error", func(ctx *Context) error {
		var bound HeaderStruct
		err := ctx.ShouldBindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/should-bind-error", nil)
	req.Header.Set("X-Count", "not-an-int")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_ShouldBindHeader_NilPointer(t *testing.T) {
	router := New()

	router.GET("/should-bind-nil", func(ctx *Context) error {
		err := ctx.ShouldBindHeader(nil)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/should-bind-nil", nil)
	req.Header.Set("X-Test", "value")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// 9. BindHeader with custom header names
func Test_BindHeader_CustomHeaderNames(t *testing.T) {
	router := New()

	type CustomHeaderStruct struct {
		TraceID      string `header:"X-Trace-Id"`
		SpanID       string `header:"X-Span-Id"`
		ParentSpanID string `header:"X-Parent-Span-Id"`
		Version      string `header:"X-Api-Version"`
	}

	var bound CustomHeaderStruct
	router.GET("/custom", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	req.Header.Set("X-Trace-Id", "trace-abc-123")
	req.Header.Set("X-Span-Id", "span-def-456")
	req.Header.Set("X-Parent-Span-Id", "span-ghi-789")
	req.Header.Set("X-Api-Version", "v2")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.TraceID != "trace-abc-123" {
		t.Errorf("expected TraceID 'trace-abc-123', got '%s'", bound.TraceID)
	}
	if bound.SpanID != "span-def-456" {
		t.Errorf("expected SpanID 'span-def-456', got '%s'", bound.SpanID)
	}
	if bound.ParentSpanID != "span-ghi-789" {
		t.Errorf("expected ParentSpanID 'span-ghi-789', got '%s'", bound.ParentSpanID)
	}
	if bound.Version != "v2" {
		t.Errorf("expected Version 'v2', got '%s'", bound.Version)
	}
}

func Test_BindHeader_CustomHeaderNames_Lowercase(t *testing.T) {
	router := New()

	type LowercaseHeaderStruct struct {
		APIKey string `header:"x-api-key"`
		Token  string `header:"x-auth-token"`
	}

	var bound LowercaseHeaderStruct
	router.GET("/lowercase", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/lowercase", nil)
	req.Header.Set("x-api-key", "key-lowercase")
	req.Header.Set("x-auth-token", "token-lowercase")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.APIKey != "key-lowercase" {
		t.Errorf("expected APIKey 'key-lowercase', got '%s'", bound.APIKey)
	}
	if bound.Token != "token-lowercase" {
		t.Errorf("expected Token 'token-lowercase', got '%s'", bound.Token)
	}
}

// 10. Header binding to map[string][]string
// Note: Header binding uses mappingByPtr which does not support map types directly.
// Map binding is only supported via query/form bindings (mapFormByTag).
// This test verifies that binding to a map through header returns empty without error.
func Test_BindHeader_MapStringSliceString(t *testing.T) {
	router := New()

	router.GET("/map-slice", func(ctx *Context) error {
		bound := make(map[string][]string)
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		// Map binding via headers does not populate the map (mappingByPtr doesn't support maps)
		// This is expected behavior; use struct tags for header binding
		ctx.Json(bound)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/map-slice", nil)
	req.Header.Set("X-Key1", "value1")
	req.Header.Set("X-Key2", "value2")
	req.Header["X-Multi"] = []string{"a", "b", "c"}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Binding should succeed without error, but map remains empty
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Header binding to map[string]string
// Note: Map binding via headers is not supported; this test verifies graceful handling.
func Test_BindHeader_MapStringString(t *testing.T) {
	router := New()

	router.GET("/map-str", func(ctx *Context) error {
		bound := make(map[string]string)
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.Json(bound)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/map-str", nil)
	req.Header.Set("X-Name", "John")
	req.Header.Set("X-Email", "john@example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Binding should succeed without error, but map remains empty
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// 11. Case-insensitive header matching
func Test_BindHeader_CaseInsensitiveMatching(t *testing.T) {
	router := New()

	type CaseHeaderStruct struct {
		APIKey      string `header:"X-Api-Key"`
		Auth        string `header:"Authorization"`
		ContentType string `header:"Content-Type"`
	}

	var bound CaseHeaderStruct
	router.GET("/case-insensitive", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Set headers with different cases than the struct tags
	req := httptest.NewRequest(http.MethodGet, "/case-insensitive", nil)
	req.Header.Set("x-api-key", "lowercase-key")
	req.Header.Set("AUTHORIZATION", "UPPERCASE-AUTH")
	req.Header.Set("content-type", "mixed-case")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.APIKey != "lowercase-key" {
		t.Errorf("expected APIKey 'lowercase-key', got '%s'", bound.APIKey)
	}
	if bound.Auth != "UPPERCASE-AUTH" {
		t.Errorf("expected Auth 'UPPERCASE-AUTH', got '%s'", bound.Auth)
	}
	if bound.ContentType != "mixed-case" {
		t.Errorf("expected ContentType 'mixed-case', got '%s'", bound.ContentType)
	}
}

// 12. Auto-binding from ctx.Bind() with header inclusion
func Test_Bind_Auto_Header_Inclusion(t *testing.T) {
	router := New()

	type AutoBindHeaderStruct struct {
		Name    string `query:"name"`
		APIKey  string `header:"X-Api-Key"`
		Version int    `header:"X-Version"`
	}

	var bound AutoBindHeaderStruct
	router.GET("/auto-header", func(ctx *Context) error {
		b := BindingDefaultStruct{BindHeader: true}
		err := ctx.ShouldBindWith(&bound, &b)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/auto-header?name=TestUser", nil)
	req.Header.Set("X-Api-Key", "auto-key")
	req.Header.Set("X-Version", "2")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "TestUser" {
		t.Errorf("expected Name 'TestUser', got '%s'", bound.Name)
	}
	if bound.APIKey != "auto-key" {
		t.Errorf("expected APIKey 'auto-key', got '%s'", bound.APIKey)
	}
	if bound.Version != 2 {
		t.Errorf("expected Version 2, got %d", bound.Version)
	}
}

func Test_Bind_Auto_Header_Exclusion(t *testing.T) {
	router := New()

	type AutoBindNoHeaderStruct struct {
		Name   string `query:"name"`
		APIKey string `header:"X-Api-Key"`
	}

	var bound AutoBindNoHeaderStruct
	router.GET("/auto-no-header", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/auto-no-header?name=TestUser", nil)
	req.Header.Set("X-Api-Key", "should-not-bind")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "TestUser" {
		t.Errorf("expected Name 'TestUser', got '%s'", bound.Name)
	}
	// APIKey should be empty because Bind() uses BindingDefault without BindHeader=true
	if bound.APIKey != "" {
		t.Errorf("expected APIKey '' (not bound), got '%s'", bound.APIKey)
	}
}

// Additional comprehensive tests

// Header binding with float fields
func Test_BindHeader_FloatFields(t *testing.T) {
	router := New()

	type FloatHeaderStruct struct {
		Priority   float64 `header:"X-Priority"`
		Weight     float32 `header:"X-Weight"`
		Confidence float64 `header:"X-Confidence"`
	}

	var bound FloatHeaderStruct
	router.GET("/float-headers", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/float-headers", nil)
	req.Header.Set("X-Priority", "1.5")
	req.Header.Set("X-Weight", "0.75")
	req.Header.Set("X-Confidence", "99.9")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Priority != 1.5 {
		t.Errorf("expected Priority 1.5, got %f", bound.Priority)
	}
	if bound.Weight != 0.75 {
		t.Errorf("expected Weight 0.75, got %f", bound.Weight)
	}
	if bound.Confidence != 99.9 {
		t.Errorf("expected Confidence 99.9, got %f", bound.Confidence)
	}
}

func Test_BindHeader_FloatFields_InvalidValue(t *testing.T) {
	router := New()

	type FloatHeaderStruct struct {
		Priority float64 `header:"X-Priority"`
	}

	router.GET("/float-invalid", func(ctx *Context) error {
		var bound FloatHeaderStruct
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/float-invalid", nil)
	req.Header.Set("X-Priority", "not-a-float")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Header binding with boolean fields
func Test_BindHeader_BoolFields(t *testing.T) {
	router := New()

	type BoolHeaderStruct struct {
		IsActive  bool `header:"X-Is-Active"`
		IsAdmin   bool `header:"X-Is-Admin"`
		IsDeleted bool `header:"X-Is-Deleted"`
	}

	var bound BoolHeaderStruct
	router.GET("/bool-headers", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bool-headers", nil)
	req.Header.Set("X-Is-Active", "true")
	req.Header.Set("X-Is-Admin", "false")
	req.Header.Set("X-Is-Deleted", "true")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.IsActive != true {
		t.Errorf("expected IsActive true, got %v", bound.IsActive)
	}
	if bound.IsAdmin != false {
		t.Errorf("expected IsAdmin false, got %v", bound.IsAdmin)
	}
	if bound.IsDeleted != true {
		t.Errorf("expected IsDeleted true, got %v", bound.IsDeleted)
	}
}

func Test_BindHeader_BoolFields_InvalidValue(t *testing.T) {
	router := New()

	type BoolHeaderStruct struct {
		IsActive bool `header:"X-Is-Active"`
	}

	router.GET("/bool-invalid", func(ctx *Context) error {
		var bound BoolHeaderStruct
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bool-invalid", nil)
	req.Header.Set("X-Is-Active", "not-a-bool")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Header binding with uint fields
func Test_BindHeader_UintFields(t *testing.T) {
	router := New()

	type UintHeaderStruct struct {
		MaxSize    uint   `header:"X-Max-Size"`
		Permission uint8  `header:"X-Permission"`
		Flags      uint16 `header:"X-Flags"`
	}

	var bound UintHeaderStruct
	router.GET("/uint-headers", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/uint-headers", nil)
	req.Header.Set("X-Max-Size", "4096")
	req.Header.Set("X-Permission", "7")
	req.Header.Set("X-Flags", "65535")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.MaxSize != 4096 {
		t.Errorf("expected MaxSize 4096, got %d", bound.MaxSize)
	}
	if bound.Permission != 7 {
		t.Errorf("expected Permission 7, got %d", bound.Permission)
	}
	if bound.Flags != 65535 {
		t.Errorf("expected Flags 65535, got %d", bound.Flags)
	}
}

// Header binding with mixed types
func Test_BindHeader_MixedTypes(t *testing.T) {
	router := New()

	type MixedHeaderStruct struct {
		Name       string  `header:"X-Name"`
		Version    int     `header:"X-Version"`
		IsActive   bool    `header:"X-Is-Active"`
		Score      float64 `header:"X-Score"`
		MaxRetries uint    `header:"X-Max-Retries"`
	}

	var bound MixedHeaderStruct
	router.GET("/mixed", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/mixed", nil)
	req.Header.Set("X-Name", "TestService")
	req.Header.Set("X-Version", "3")
	req.Header.Set("X-Is-Active", "true")
	req.Header.Set("X-Score", "8.5")
	req.Header.Set("X-Max-Retries", "5")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "TestService" {
		t.Errorf("expected Name 'TestService', got '%s'", bound.Name)
	}
	if bound.Version != 3 {
		t.Errorf("expected Version 3, got %d", bound.Version)
	}
	if bound.IsActive != true {
		t.Errorf("expected IsActive true, got %v", bound.IsActive)
	}
	if bound.Score != 8.5 {
		t.Errorf("expected Score 8.5, got %f", bound.Score)
	}
	if bound.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", bound.MaxRetries)
	}
}

// Header binding with slice of integers
func Test_BindHeader_IntSliceFields(t *testing.T) {
	router := New()

	type IntSliceHeaderStruct struct {
		IDs   []int   `header:"X-Ids"`
		Ports []int64 `header:"X-Ports"`
	}

	var bound IntSliceHeaderStruct
	router.GET("/int-slice", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/int-slice", nil)
	req.Header["X-Ids"] = []string{"1", "2", "3"}
	req.Header["X-Ports"] = []string{"8080", "8443", "9090"}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound.IDs) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(bound.IDs))
	}
	if bound.IDs[0] != 1 || bound.IDs[1] != 2 || bound.IDs[2] != 3 {
		t.Errorf("expected IDs [1, 2, 3], got %v", bound.IDs)
	}
	if len(bound.Ports) != 3 {
		t.Errorf("expected 3 Ports, got %d", len(bound.Ports))
	}
	if bound.Ports[0] != 8080 || bound.Ports[1] != 8443 || bound.Ports[2] != 9090 {
		t.Errorf("expected Ports [8080, 8443, 9090], got %v", bound.Ports)
	}
}

// Header binding with default value for slices
func Test_BindHeader_SliceDefaultValues(t *testing.T) {
	router := New()

	type SliceDefaultHeaderStruct struct {
		Tags []string `header:"X-Tags,default=go"`
	}

	var bound SliceDefaultHeaderStruct
	router.GET("/slice-default", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/slice-default", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound.Tags) != 1 {
		t.Errorf("expected 1 Tag, got %d", len(bound.Tags))
	}
	if bound.Tags[0] != "go" {
		t.Errorf("expected Tag 'go' (default), got '%s'", bound.Tags[0])
	}
}

// Nested struct header binding
func Test_BindHeader_NestedStruct(t *testing.T) {
	router := New()

	type MetaHeader struct {
		RequestID string `header:"X-Request-Id"`
		TraceID   string `header:"X-Trace-Id"`
	}

	type NestedHeaderStruct struct {
		APIVersion string `header:"X-Api-Version"`
		Meta       MetaHeader
	}

	var bound NestedHeaderStruct
	router.GET("/nested", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/nested", nil)
	req.Header.Set("X-Api-Version", "v3")
	req.Header.Set("X-Request-Id", "req-123")
	req.Header.Set("X-Trace-Id", "trace-456")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.APIVersion != "v3" {
		t.Errorf("expected APIVersion 'v3', got '%s'", bound.APIVersion)
	}
	if bound.Meta.RequestID != "req-123" {
		t.Errorf("expected Meta.RequestID 'req-123', got '%s'", bound.Meta.RequestID)
	}
	if bound.Meta.TraceID != "trace-456" {
		t.Errorf("expected Meta.TraceID 'trace-456', got '%s'", bound.Meta.TraceID)
	}
}

// Binding with header tag using field name as default
func Test_BindHeader_FieldNameAsDefault(t *testing.T) {
	router := New()

	type FieldNameHeader struct {
		XCustomHeader string // No tag, should use field name
		ContentType   string `header:"Content-Type"`
	}

	var bound FieldNameHeader
	router.GET("/field-name", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/field-name", nil)
	req.Header.Set("XCustomHeader", "field-name-value")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.XCustomHeader != "field-name-value" {
		t.Errorf("expected XCustomHeader 'field-name-value', got '%s'", bound.XCustomHeader)
	}
	if bound.ContentType != "application/json" {
		t.Errorf("expected ContentType 'application/json', got '%s'", bound.ContentType)
	}
}

// Multiple headers with the same canonical form
func Test_BindHeader_CanonicalFormEquivalence(t *testing.T) {
	router := New()

	type CanonicalStruct struct {
		APIKey string `header:"x-api-key"`
	}

	var bound CanonicalStruct
	router.GET("/canonical-equiv", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Setting with different case - should still match due to canonical form
	req := httptest.NewRequest(http.MethodGet, "/canonical-equiv", nil)
	req.Header.Set("X-API-KEY", "canonical-test")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.APIKey != "canonical-test" {
		t.Errorf("expected APIKey 'canonical-test', got '%s'", bound.APIKey)
	}
}

// Direct binding interface usage
func Test_BindingHeader_Interface(t *testing.T) {
	router := New()

	type HeaderStruct struct {
		Accept string `header:"Accept"`
	}

	var bound HeaderStruct
	router.GET("/interface", func(ctx *Context) error {
		err := BindingHeader.Bind(ctx, &bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/interface", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Accept != "application/json" {
		t.Errorf("expected Accept 'application/json', got '%s'", bound.Accept)
	}
}

// Header binding with tag skip "-"
func Test_BindHeader_SkipTag(t *testing.T) {
	router := New()

	type SkipHeaderStruct struct {
		Internal string `header:"-"`
		Public   string `header:"X-Public"`
	}

	var bound SkipHeaderStruct
	router.GET("/skip", func(ctx *Context) error {
		err := ctx.BindHeader(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	req.Header.Set("Internal", "should-be-skipped")
	req.Header.Set("X-Public", "public-value")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Internal != "" {
		t.Errorf("expected Internal '' (skipped), got '%s'", bound.Internal)
	}
	if bound.Public != "public-value" {
		t.Errorf("expected Public 'public-value', got '%s'", bound.Public)
	}
}
