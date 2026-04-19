package chain

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_BindXML_Success(t *testing.T) {
	router := New()

	type User struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<user><name>John</name><email>john@example.com</email></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
	if bound.Email != "john@example.com" {
		t.Errorf("expected Email 'john@example.com', got '%s'", bound.Email)
	}
}

func Test_BindXML_MalformedXML(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var obj map[string]any
		err := ctx.BindXML(&obj)
		if err != nil {
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(`<not-closed>`)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindXML_EmptyBody(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var obj struct{}
		err := ctx.BindXML(&obj)
		if err != nil {
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindXML_NestedStruct(t *testing.T) {
	router := New()

	type Address struct {
		Street string `xml:"street"`
		City   string `xml:"city"`
	}

	type Person struct {
		Name    string  `xml:"name"`
		Address Address `xml:"address"`
	}

	var bound Person
	router.POST("/person", func(ctx *Context) error {
		err := ctx.BindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<person><name>Jane</name><address><street>123 Main St</street><city>NYC</city></address></person>`
	req := httptest.NewRequest(http.MethodPost, "/person", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "Jane" {
		t.Errorf("expected Name 'Jane', got '%s'", bound.Name)
	}
	if bound.Address.Street != "123 Main St" {
		t.Errorf("expected Street '123 Main St', got '%s'", bound.Address.Street)
	}
	if bound.Address.City != "NYC" {
		t.Errorf("expected City 'NYC', got '%s'", bound.Address.City)
	}
}

func Test_BindXML_Slice(t *testing.T) {
	router := New()

	type Item struct {
		Name string `xml:"name"`
	}

	type Items struct {
		List []Item `xml:"item"`
	}

	var bound Items
	router.POST("/items", func(ctx *Context) error {
		err := ctx.BindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<items><item><name>Item1</name></item><item><name>Item2</name></item><item><name>Item3</name></item></items>`
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound.List) != 3 {
		t.Errorf("expected 3 items, got %d", len(bound.List))
	}
}

func Test_BindXML_UnknownElementsIgnored(t *testing.T) {
	router := New()

	type User struct {
		Name string `xml:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// XML with extra unknown elements
	body := `<user><name>John</name><age>30</age><email>john@example.com</email></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindXML_WithAttributes(t *testing.T) {
	router := New()

	type Item struct {
		ID    string `xml:"id,attr"`
		Name  string `xml:"name"`
		Price string `xml:"price,attr"`
	}

	var bound Item
	router.POST("/item", func(ctx *Context) error {
		err := ctx.BindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<item id="123" price="9.99"><name>Widget</name></item>`
	req := httptest.NewRequest(http.MethodPost, "/item", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "123" {
		t.Errorf("expected ID '123', got '%s'", bound.ID)
	}
	if bound.Name != "Widget" {
		t.Errorf("expected Name 'Widget', got '%s'", bound.Name)
	}
	if bound.Price != "9.99" {
		t.Errorf("expected Price '9.99', got '%s'", bound.Price)
	}
}

func Test_ShouldBindXML_Success(t *testing.T) {
	router := New()

	type User struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.ShouldBindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<user><name>John</name><email>john@example.com</email></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
	if bound.Email != "john@example.com" {
		t.Errorf("expected Email 'john@example.com', got '%s'", bound.Email)
	}
}

func Test_ShouldBindXML_Error(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var obj map[string]any
		err := ctx.ShouldBindXML(&obj)
		if err != nil {
			// ShouldBindXML should return error without writing response
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(`not xml`)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindXML_TextXMLContentType(t *testing.T) {
	router := New()

	type User struct {
		Name string `xml:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindXML(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<user><name>John</name></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "text/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindXML_Auto_From_Default(t *testing.T) {
	router := New()

	type User struct {
		Name string `xml:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<user><name>John</name></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindXML_Auto_From_Default_TextXML(t *testing.T) {
	router := New()

	type User struct {
		Name string `xml:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<user><name>John</name></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "text/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindingXML_Direct(t *testing.T) {
	router := New()

	type User struct {
		Name string `xml:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := BindingXML.Bind(ctx, &bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `<user><name>John</name></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindXML_BodyReadOnce(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		// Read body twice - should work due to caching
		body1, err := ctx.BodyBytes()
		if err != nil {
			return err
		}

		body2, err := ctx.BodyBytes()
		if err != nil {
			return err
		}

		if !bytes.Equal(body1, body2) {
			ctx.Error("body mismatch", http.StatusInternalServerError)
			return nil
		}

		ctx.OK()
		return nil
	})

	body := `<user><name>John</name></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
