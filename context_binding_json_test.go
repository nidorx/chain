package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_BindJSON_Success(t *testing.T) {
	router := New()

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := User{Name: "John", Email: "john@example.com", Age: 30}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
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
	if bound.Age != 30 {
		t.Errorf("expected Age 30, got %d", bound.Age)
	}
}

func Test_BindJSON_InvalidJSON(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var obj map[string]any
		err := ctx.BindJSON(&obj)
		if err != nil {
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(`{invalid json}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindJSON_EmptyBody(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var obj map[string]any
		err := ctx.BindJSON(&obj)
		if err != nil {
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindJSON_NilPointer(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindJSON(nil)
		if err != nil {
			return err
		}
		ctx.OK()
		return nil
	})

	body := map[string]string{"name": "John"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// json.Decode(nil) returns an error
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindJSON_NestedStruct(t *testing.T) {
	router := New()

	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	var bound Person
	router.POST("/person", func(ctx *Context) error {
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := Person{
		Name:    "Jane",
		Address: Address{Street: "123 Main St", City: "NYC"},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/person", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
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

func Test_BindJSON_Slice(t *testing.T) {
	router := New()

	var bound []string
	router.POST("/items", func(ctx *Context) error {
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := []string{"item1", "item2", "item3"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound) != 3 {
		t.Errorf("expected 3 items, got %d", len(bound))
	}
}

func Test_BindJSON_WithExtraFields(t *testing.T) {
	router := New()

	type User struct {
		Name string `json:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// JSON with extra fields not in struct
	body := `{"name": "John", "age": 30, "email": "john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindJSON_DisallowUnknownFields(t *testing.T) {
	router := New()

	// Enable strict field checking
	orig := EnableDecoderDisallowUnknownFields
	EnableDecoderDisallowUnknownFields = true
	defer func() { EnableDecoderDisallowUnknownFields = orig }()

	type User struct {
		Name string `json:"name"`
	}

	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `{"name": "John", "unknown_field": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindJSON_UseNumber(t *testing.T) {
	router := New()

	// Enable UseNumber
	orig := EnableDecoderUseNumber
	EnableDecoderUseNumber = true
	defer func() { EnableDecoderUseNumber = orig }()

	router.POST("/data", func(ctx *Context) error {
		var bound map[string]any
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		// With UseNumber, numbers should be json.Number, not float64
		if _, ok := bound["id"].(json.Number); !ok {
			ctx.Error("expected json.Number type", http.StatusInternalServerError)
			return errors.New("expected json.Number")
		}
		ctx.OK()
		return nil
	})

	body := `{"id": 12345, "name": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/data", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_ShouldBindJSON_Success(t *testing.T) {
	router := New()

	type User struct {
		Name string `json:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.ShouldBindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := User{Name: "John"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_ShouldBindJSON_Invalid(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var obj map[string]any
		err := ctx.ShouldBindJSON(&obj)
		if err != nil {
			// ShouldBindJSON should return error without writing response
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(`invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindJSON_BodyReadOnce(t *testing.T) {
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
			return errors.New("body mismatch")
		}

		ctx.OK()
		return nil
	})

	body := `{"name": "John"}`
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_BindJSON_Map(t *testing.T) {
	router := New()

	var bound map[string]string
	router.POST("/data", func(ctx *Context) error {
		err := ctx.BindJSON(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := `{"key1": "value1", "key2": "value2"}`
	req := httptest.NewRequest(http.MethodPost, "/data", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound["key1"] != "value1" {
		t.Errorf("expected key1='value1', got '%s'", bound["key1"])
	}
	if bound["key2"] != "value2" {
		t.Errorf("expected key2='value2', got '%s'", bound["key2"])
	}
}

func Test_Bind_JSON_Auto_From_Default(t *testing.T) {
	router := New()

	type User struct {
		Name string `json:"name"`
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

	body := User{Name: "John"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}
