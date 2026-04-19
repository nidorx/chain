package chain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func PerformRequest(router *Router, method string, url string) *httptest.ResponseRecorder {
	r, _ := http.NewRequest(method, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

type handlerStructMid struct {
	onCall func()
}

func (h handlerStructMid) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.onCall()
}

func Test_Middleware_Signatures(t *testing.T) {
	signature := ""
	router := New()
	if _, err := router.Use(func() {
		signature += "A"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func() error {
		signature += "B"
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(c *Context) {
		signature += "C"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(c *Context) error {
		signature += "D"
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(next func() error) {
		signature += "E"
		next()
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(next func() error) error {
		signature += "F"
		return next()
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(c *Context, next func() error) {
		signature += "G"
		next()
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(c *Context, next func() error) error {
		signature += "H"
		return next()
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(handlerStructMid{onCall: func() {
		signature += "I"
	}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		signature += "J"
	})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := router.Use(func(string2 string) {
		signature += " XXX "
	})
	if err == nil {
		t.Errorf("no error for invalid middleware")
	}
	router.GET("/", func(c *Context) error {
		signature += "X"
		return nil
	})

	w := PerformRequest(router, "GET", "/")

	if w.Code != http.StatusOK {
		t.Errorf("router.Use() failed: Invalid Code\n   actual: %v\n expected: %v", w.Code, http.StatusOK)
	}

	expected := "ABCDEFGHIJX"
	if signature != expected {
		t.Errorf("router.Use() failed: Invalid Execution Order\n   actual: %v\n expected: %v", signature, expected)
	}
}

func Test_Middleware_GeneralCase(t *testing.T) {
	signature := ""
	router := New()
	if _, err := router.Use(func(c *Context, next func() error) error {
		signature += "A"
		next()
		next()
		signature += "B"
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(c *Context) {
		signature += "C"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	router.GET("/", func(c *Context) error {
		signature += "D"
		return nil
	})

	w := PerformRequest(router, "GET", "/")

	if w.Code != http.StatusOK {
		t.Errorf("router.Use() failed: Invalid Code\n   actual: %v\n expected: %v", w.Code, http.StatusOK)
	}

	if signature != "ACDB" {
		t.Errorf("router.Use() failed: Invalid Execution Order\n   actual: %v\n expected: %v", signature, "ACDB")
	}
}

func Test_Middleware_NotFound(t *testing.T) {
	signature := ""
	router := New()
	if _, err := router.Use(func(c *Context, next func() error) {
		signature += "A"
		next()
		signature += "B"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(c *Context, next func() error) {
		signature += "C"
		next()
		next()
		next()
		next()
		signature += "D"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		signature += " X "
		http.NotFound(w, req)
	})
	w := PerformRequest(router, "GET", "/")

	if w.Code != http.StatusNotFound {
		t.Errorf("router.Use() failed: Invalid Code\n   actual: %v\n expected: %v", w.Code, http.StatusNotFound)
	}

	if signature != " X " {
		t.Errorf("router.Use() failed: Invalid Execution Order\n   actual: %v\n expected: %v", signature, " X ")
	}
}

func Test_Middleware_Abort(t *testing.T) {
	signature := ""
	router := New()
	if _, err := router.Use(func() {
		signature += "A"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := router.Use(func(ctx *Context, next func() error) {
		signature += "C"

		ctx.WriteHeader(http.StatusUnauthorized)
		// dont call next

		signature += "D"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	router.GET("/", func(ctx *Context) {
		signature += " X "
	})

	w := PerformRequest(router, "GET", "/")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("router.Use() failed: Invalid Code\n   actual: %v\n expected: %v", w.Code, http.StatusUnauthorized)
	}

	if signature != "ACD" {
		t.Errorf("router.Use() failed: Invalid Execution Order\n   actual: %v\n expected: %v", signature, "ACD")
	}
}
