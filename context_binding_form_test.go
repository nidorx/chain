package chain

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ============================================================
// 1. Successful form binding with POST x-www-form-urlencoded data
// ============================================================

func Test_BindForm_Success(t *testing.T) {
	router := New()

	type User struct {
		Name  string `form:"name"`
		Email string `form:"email"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&email=john%40example.com"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

// ============================================================
// 2. Successful form post binding
// ============================================================

func Test_BindFormPost_Success(t *testing.T) {
	router := New()

	type Login struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	var bound Login
	router.POST("/login", func(ctx *Context) error {
		err := ctx.BindFormPost(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "username=admin&password=secret123"
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Username != "admin" {
		t.Errorf("expected Username 'admin', got '%s'", bound.Username)
	}
	if bound.Password != "secret123" {
		t.Errorf("expected Password 'secret123', got '%s'", bound.Password)
	}
}

// ============================================================
// 3. Successful multipart form binding with file upload
// ============================================================

func Test_BindFormMultipart_Success(t *testing.T) {
	router := New()

	type UploadForm struct {
		Title       string               `form:"title"`
		Description string               `form:"description"`
		File        multipart.FileHeader `form:"file"`
	}

	var bound UploadForm
	router.POST("/upload", func(ctx *Context) error {
		err := ctx.BindFormMultipart(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("title", "My Document")
	writer.WriteField("description", "A test file")
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("hello world"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Title != "My Document" {
		t.Errorf("expected Title 'My Document', got '%s'", bound.Title)
	}
	if bound.Description != "A test file" {
		t.Errorf("expected Description 'A test file', got '%s'", bound.Description)
	}
	if bound.File.Filename != "test.txt" {
		t.Errorf("expected File.Filename 'test.txt', got '%s'", bound.File.Filename)
	}
}

// ============================================================
// 4. Form binding with various field types (string, int, bool, float, slice)
// ============================================================

func Test_BindForm_VariousFieldTypes(t *testing.T) {
	router := New()

	type AllTypes struct {
		Name   string   `form:"name"`
		Age    int      `form:"age"`
		Active bool     `form:"active"`
		Score  float64  `form:"score"`
		Rating float32  `form:"rating"`
		Count  int64    `form:"count"`
		UCount uint64   `form:"ucount"`
		Tags   []string `form:"tags"`
		Scores []int    `form:"scores"`
	}

	var bound AllTypes
	router.POST("/data", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&age=30&active=true&score=9.5&rating=8.7&count=1000&ucount=500&tags=go&tags=rust&tags=python&scores=1&scores=2&scores=3"
	req := httptest.NewRequest(http.MethodPost, "/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
	if bound.Age != 30 {
		t.Errorf("expected Age 30, got %d", bound.Age)
	}
	if bound.Active != true {
		t.Errorf("expected Active true, got %v", bound.Active)
	}
	if bound.Score != 9.5 {
		t.Errorf("expected Score 9.5, got %f", bound.Score)
	}
	if bound.Rating != 8.7 {
		t.Errorf("expected Rating 8.7, got %f", bound.Rating)
	}
	if bound.Count != 1000 {
		t.Errorf("expected Count 1000, got %d", bound.Count)
	}
	if bound.UCount != 500 {
		t.Errorf("expected UCount 500, got %d", bound.UCount)
	}
	if len(bound.Tags) != 3 {
		t.Errorf("expected 3 Tags, got %d", len(bound.Tags))
	}
	if bound.Tags[0] != "go" || bound.Tags[1] != "rust" || bound.Tags[2] != "python" {
		t.Errorf("expected Tags [go rust python], got %v", bound.Tags)
	}
	if len(bound.Scores) != 3 {
		t.Errorf("expected 3 Scores, got %d", len(bound.Scores))
	}
	if bound.Scores[0] != 1 || bound.Scores[1] != 2 || bound.Scores[2] != 3 {
		t.Errorf("expected Scores [1 2 3], got %v", bound.Scores)
	}
}

func Test_BindForm_IntTypes(t *testing.T) {
	router := New()

	type IntTypes struct {
		I8  int8   `form:"i8"`
		I16 int16  `form:"i16"`
		I32 int32  `form:"i32"`
		U8  uint8  `form:"u8"`
		U16 uint16 `form:"u16"`
		U32 uint32 `form:"u32"`
	}

	var bound IntTypes
	router.POST("/inttypes", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "i8=8&i16=16&i32=32&u8=80&u16=160&u32=320"
	req := httptest.NewRequest(http.MethodPost, "/inttypes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.I8 != 8 {
		t.Errorf("expected I8=8, got %d", bound.I8)
	}
	if bound.I16 != 16 {
		t.Errorf("expected I16=16, got %d", bound.I16)
	}
	if bound.I32 != 32 {
		t.Errorf("expected I32=32, got %d", bound.I32)
	}
	if bound.U8 != 80 {
		t.Errorf("expected U8=80, got %d", bound.U8)
	}
	if bound.U16 != 160 {
		t.Errorf("expected U16=160, got %d", bound.U16)
	}
	if bound.U32 != 320 {
		t.Errorf("expected U32=320, got %d", bound.U32)
	}
}

func Test_BindForm_Duration(t *testing.T) {
	router := New()

	type Timeout struct {
		Timeout time.Duration `form:"timeout"`
	}

	var bound Timeout
	router.POST("/timeout", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "timeout=5s"
	req := httptest.NewRequest(http.MethodPost, "/timeout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Timeout != 5*time.Second {
		t.Errorf("expected Timeout 5s, got %v", bound.Timeout)
	}
}

// ============================================================
// 5. Form binding with default values via tag `form:"field,default=value"`
// ============================================================

func Test_BindForm_DefaultValues(t *testing.T) {
	router := New()

	type Config struct {
		Name    string  `form:"name,default=anonymous"`
		Page    int     `form:"page,default=1"`
		Limit   int     `form:"limit,default=20"`
		Enabled bool    `form:"enabled,default=true"`
		Ratio   float64 `form:"ratio,default=1.5"`
	}

	var bound Config
	router.POST("/config", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Send only name; rest should use defaults
	body := "name=custom"
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "custom" {
		t.Errorf("expected Name 'custom', got '%s'", bound.Name)
	}
	if bound.Page != 1 {
		t.Errorf("expected Page 1, got %d", bound.Page)
	}
	if bound.Limit != 20 {
		t.Errorf("expected Limit 20, got %d", bound.Limit)
	}
	if bound.Enabled != true {
		t.Errorf("expected Enabled true, got %v", bound.Enabled)
	}
	if bound.Ratio != 1.5 {
		t.Errorf("expected Ratio 1.5, got %f", bound.Ratio)
	}
}

func Test_BindForm_DefaultValues_Override(t *testing.T) {
	router := New()

	type Config struct {
		Page  int `form:"page,default=1"`
		Limit int `form:"limit,default=20"`
	}

	var bound Config
	router.POST("/config", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Override all defaults
	body := "page=5&limit=50"
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Page != 5 {
		t.Errorf("expected Page 5, got %d", bound.Page)
	}
	if bound.Limit != 50 {
		t.Errorf("expected Limit 50, got %d", bound.Limit)
	}
}

func Test_BindForm_DefaultValues_Slice(t *testing.T) {
	router := New()

	type Query struct {
		Tags []string `form:"tags,default=general"`
	}

	var bound Query
	router.POST("/query", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// No tags provided; should use default
	body := "sort=date"
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound.Tags) != 1 || bound.Tags[0] != "general" {
		t.Errorf("expected Tags ['general'], got %v", bound.Tags)
	}
}

// ============================================================
// 6. Form binding with field skipping via tag `form:"-"`
// ============================================================

func Test_BindForm_FieldSkipping(t *testing.T) {
	router := New()

	type User struct {
		Name     string `form:"name"`
		Secret   string `form:"-"`
		Password string `form:"password"`
	}

	var bound User
	bound.Secret = "pre-existing"
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Include "secret" in form data; it should be ignored
	body := "name=John&secret=should-not-bind&password=pass123"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
	if bound.Secret != "pre-existing" {
		t.Errorf("expected Secret unchanged, got '%s'", bound.Secret)
	}
	if bound.Password != "pass123" {
		t.Errorf("expected Password 'pass123', got '%s'", bound.Password)
	}
}

// ============================================================
// 7. Empty form data
// ============================================================

func Test_BindForm_EmptyData(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
	}

	var bound User
	bound.Name = "initial"
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	// Empty form data does not overwrite existing values
	if bound.Name != "initial" {
		t.Errorf("expected Name 'initial', got '%s'", bound.Name)
	}
}

func Test_BindFormPost_EmptyData(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
	}

	var bound User
	bound.Name = "initial"
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindFormPost(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "initial" {
		t.Errorf("expected Name 'initial', got '%s'", bound.Name)
	}
}

// ============================================================
// 8. Malformed form data
// ============================================================

func Test_BindForm_MalformedData(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
	}

	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// %ZZ is an invalid percent encoding
	body := "name=%ZZinvalid"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindFormPost_MalformedData(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var bound struct {
			Name string `form:"name"`
		}
		err := ctx.BindFormPost(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=%ZZinvalid"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindForm_InvalidInt(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&age=notanumber"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindForm_InvalidBool(t *testing.T) {
	router := New()

	type User struct {
		Name   string `form:"name"`
		Active bool   `form:"active"`
	}

	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&active=maybe"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindForm_InvalidFloat(t *testing.T) {
	router := New()

	type User struct {
		Name  string  `form:"name"`
		Score float64 `form:"score"`
	}

	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&score=notafloat"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============================================================
// 9. Multipart form with multiple files
// ============================================================

func Test_BindFormMultipart_MultipleFiles(t *testing.T) {
	router := New()

	type MultiUpload struct {
		Title string                 `form:"title"`
		Files []multipart.FileHeader `form:"files"`
	}

	var bound MultiUpload
	router.POST("/upload", func(ctx *Context) error {
		err := ctx.BindFormMultipart(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("title", "Multi-file upload")

	// Create three files
	for i := 1; i <= 3; i++ {
		part, _ := writer.CreateFormFile("files", "file"+string(rune('0'+i))+".txt")
		part.Write([]byte("content of file " + string(rune('0'+i))))
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Title != "Multi-file upload" {
		t.Errorf("expected Title 'Multi-file upload', got '%s'", bound.Title)
	}
	if len(bound.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(bound.Files))
	}
	if bound.Files[0].Filename != "file1.txt" {
		t.Errorf("expected first file 'file1.txt', got '%s'", bound.Files[0].Filename)
	}
	if bound.Files[1].Filename != "file2.txt" {
		t.Errorf("expected second file 'file2.txt', got '%s'", bound.Files[1].Filename)
	}
	if bound.Files[2].Filename != "file3.txt" {
		t.Errorf("expected third file 'file3.txt', got '%s'", bound.Files[2].Filename)
	}
}

func Test_BindFormMultipart_PointerFileHeader(t *testing.T) {
	router := New()

	type Upload struct {
		Title string                `form:"title"`
		File  *multipart.FileHeader `form:"file"`
	}

	var bound Upload
	router.POST("/upload", func(ctx *Context) error {
		err := ctx.BindFormMultipart(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("title", "Pointer file test")
	part, _ := writer.CreateFormFile("file", "ptr.txt")
	part.Write([]byte("pointer content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.File == nil {
		t.Fatal("expected File to be non-nil")
	}
	if bound.File.Filename != "ptr.txt" {
		t.Errorf("expected Filename 'ptr.txt', got '%s'", bound.File.Filename)
	}
}

func Test_BindFormMultipart_MixedValuesAndFiles(t *testing.T) {
	router := New()

	type MixedForm struct {
		Name   string               `form:"name"`
		Age    int                  `form:"age"`
		Active bool                 `form:"active"`
		Avatar multipart.FileHeader `form:"avatar"`
		Notes  string               `form:"notes"`
	}

	var bound MixedForm
	router.POST("/mixed", func(ctx *Context) error {
		err := ctx.BindFormMultipart(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("name", "Alice")
	writer.WriteField("age", "28")
	writer.WriteField("active", "true")
	part, _ := writer.CreateFormFile("avatar", "photo.png")
	part.Write([]byte("\x89PNG..."))
	writer.WriteField("notes", "Hello from Alice")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/mixed", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got '%s'", bound.Name)
	}
	if bound.Age != 28 {
		t.Errorf("expected Age 28, got %d", bound.Age)
	}
	if bound.Active != true {
		t.Errorf("expected Active true, got %v", bound.Active)
	}
	if bound.Avatar.Filename != "photo.png" {
		t.Errorf("expected Avatar filename 'photo.png', got '%s'", bound.Avatar.Filename)
	}
	if bound.Notes != "Hello from Alice" {
		t.Errorf("expected Notes 'Hello from Alice', got '%s'", bound.Notes)
	}
}

// ============================================================
// 10. ShouldBindForm success and error cases
// ============================================================

func Test_ShouldBindForm_Success(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.ShouldBindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_ShouldBindForm_Error(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	var statusCode int
	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.ShouldBindForm(&bound)
		if err != nil {
			// ShouldBindForm returns error without writing response
			statusCode = http.StatusBadRequest
			ctx.Error(err.Error(), statusCode)
			return err
		}
		statusCode = http.StatusOK
		ctx.OK()
		return nil
	})

	body := "name=John&age=invalid"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if statusCode != http.StatusBadRequest {
		t.Errorf("expected statusCode %d, got %d", http.StatusBadRequest, statusCode)
	}
}

func Test_ShouldBindFormPost_Error(t *testing.T) {
	router := New()

	router.POST("/user", func(ctx *Context) error {
		var bound struct {
			Age int `form:"age"`
		}
		err := ctx.ShouldBindFormPost(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "age=notint"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_ShouldBindFormMultipart_Error(t *testing.T) {
	router := New()

	router.POST("/upload", func(ctx *Context) error {
		var bound struct {
			Name string `form:"name"`
		}
		err := ctx.ShouldBindFormMultipart(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Not a multipart request
	body := "name=John"
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============================================================
// 11. BindForm with time.Time fields using time_format tag
// ============================================================

func Test_BindForm_TimeField_RFC3339(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		CreatedAt time.Time `form:"created_at"`
	}

	var bound Event
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=Birthday&created_at=2024-01-15T10:30:00Z"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "Birthday" {
		t.Errorf("expected Name 'Birthday', got '%s'", bound.Name)
	}
	expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !bound.CreatedAt.Equal(expected) {
		t.Errorf("expected CreatedAt %v, got %v", expected, bound.CreatedAt)
	}
}

func Test_BindForm_TimeField_CustomFormat(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		EventDate time.Time `form:"event_date" time_format:"2006-01-02"`
	}

	var bound Event
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=Conference&event_date=2024-06-15"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "Conference" {
		t.Errorf("expected Name 'Conference', got '%s'", bound.Name)
	}
	expected := time.Date(2024, 6, 15, 0, 0, 0, 0, time.Local)
	if !bound.EventDate.Equal(expected) {
		t.Errorf("expected EventDate %v, got %v", expected, bound.EventDate)
	}
}

func Test_BindForm_TimeField_DateTimeFormat(t *testing.T) {
	router := New()

	type Event struct {
		Name     string    `form:"name"`
		StartsAt time.Time `form:"starts_at" time_format:"2006-01-02 15:04:05"`
	}

	var bound Event
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=Meeting&starts_at=2024-03-20 09:00:00"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	expected := time.Date(2024, 3, 20, 9, 0, 0, 0, time.Local)
	if !bound.StartsAt.Equal(expected) {
		t.Errorf("expected StartsAt %v, got %v", expected, bound.StartsAt)
	}
}

func Test_BindForm_TimeField_InvalidFormat(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		CreatedAt time.Time `form:"created_at" time_format:"2006-01-02"`
	}

	router.POST("/event", func(ctx *Context) error {
		var bound Event
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Invalid date format for the specified time_format
	body := "name=BadEvent&created_at=not-a-date"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindForm_TimeField_Empty(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		CreatedAt time.Time `form:"created_at" time_format:"2006-01-02"`
	}

	var bound Event
	bound.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Empty time value should reset to zero time
	body := "name=Test&created_at="
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !bound.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt to be zero time, got %v", bound.CreatedAt)
	}
}

func Test_BindForm_TimeField_Unix(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		Timestamp time.Time `form:"timestamp" time_format:"unix"`
	}

	var bound Event
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=UnixTime&timestamp=1700000000"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	expected := time.Unix(1700000000, 0)
	if !bound.Timestamp.Equal(expected) {
		t.Errorf("expected Timestamp %v, got %v", expected, bound.Timestamp)
	}
}

func Test_BindForm_TimeField_UnixNano(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		Timestamp time.Time `form:"timestamp" time_format:"unixnano"`
	}

	var bound Event
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=UnixNano&timestamp=1700000000000000000"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	expected := time.Unix(1700000000, 0)
	if !bound.Timestamp.Equal(expected) {
		t.Errorf("expected Timestamp %v, got %v", expected, bound.Timestamp)
	}
}

func Test_BindForm_TimeField_UTC(t *testing.T) {
	router := New()

	type Event struct {
		Name      string    `form:"name"`
		CreatedAt time.Time `form:"created_at" time_format:"2006-01-02 15:04:05" time_utc:"true"`
	}

	var bound Event
	router.POST("/event", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=UTCEvent&created_at=2024-06-15 12:00:00"
	req := httptest.NewRequest(http.MethodPost, "/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	// Verify the parsed time is in UTC
	if bound.CreatedAt.Location() != time.UTC {
		t.Errorf("expected CreatedAt in UTC, got %v", bound.CreatedAt.Location())
	}
}

// ============================================================
// 12. Auto-binding from ctx.Bind() with application/x-www-form-urlencoded content type
// ============================================================

func Test_Bind_Auto_FormEncoded(t *testing.T) {
	router := New()

	type User struct {
		Name  string `form:"name"`
		Email string `form:"email"`
		Age   int    `form:"age"`
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

	body := "name=Alice&email=alice%40example.com&age=25"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got '%s'", bound.Name)
	}
	if bound.Email != "alice@example.com" {
		t.Errorf("expected Email 'alice@example.com', got '%s'", bound.Email)
	}
	if bound.Age != 25 {
		t.Errorf("expected Age 25, got %d", bound.Age)
	}
}

func Test_ShouldBind_Auto_FormEncoded(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.ShouldBind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=Bob"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "Bob" {
		t.Errorf("expected Name 'Bob', got '%s'", bound.Name)
	}
}

func Test_Bind_Auto_FormEncoded_InvalidInt(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	router.POST("/user", func(ctx *Context) error {
		var bound User
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=Bob&age=notanumber"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============================================================
// Additional tests: nested structs, pointer fields, map[string]string
// ============================================================

func Test_BindForm_NestedStruct(t *testing.T) {
	router := New()

	type Address struct {
		City    string `form:"city"`
		Country string `form:"country"`
	}

	type User struct {
		Name    string  `form:"name"`
		Address Address `form:"address"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&city=NYC&country=USA"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
	if bound.Address.City != "NYC" {
		t.Errorf("expected City 'NYC', got '%s'", bound.Address.City)
	}
	if bound.Address.Country != "USA" {
		t.Errorf("expected Country 'USA', got '%s'", bound.Address.Country)
	}
}

func Test_BindForm_PointerField(t *testing.T) {
	router := New()

	type User struct {
		Name string `form:"name"`
		Age  *int   `form:"age"`
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=John&age=30"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
	if bound.Age == nil {
		t.Fatal("expected Age to be non-nil")
	}
	if *bound.Age != 30 {
		t.Errorf("expected Age 30, got %d", *bound.Age)
	}
}

func Test_BindForm_MapStringString(t *testing.T) {
	router := New()

	bound := make(map[string]string)
	router.POST("/data", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "key1=value1&key2=value2"
	req := httptest.NewRequest(http.MethodPost, "/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound["key1"] != "value1" {
		t.Errorf("expected key1='value1', got '%s'", bound["key1"])
	}
	if bound["key2"] != "value2" {
		t.Errorf("expected key2='value2', got '%s'", bound["key2"])
	}
}

func Test_BindForm_MapStringSlice(t *testing.T) {
	router := New()

	bound := make(map[string][]string)
	router.POST("/data", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "tags=go&tags=rust&name=test"
	req := httptest.NewRequest(http.MethodPost, "/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if len(bound["tags"]) != 2 {
		t.Errorf("expected 2 tags, got %d", len(bound["tags"]))
	}
	if bound["name"][0] != "test" {
		t.Errorf("expected name[0]='test', got '%s'", bound["name"][0])
	}
}

func Test_BindForm_DefaultFieldName_FromStructField(t *testing.T) {
	router := New()

	type User struct {
		Name string // no form tag; should use field name
	}

	var bound User
	router.POST("/user", func(ctx *Context) error {
		err := ctx.BindForm(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Uses struct field name "Name" as the form key
	body := "Name=John"
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John" {
		t.Errorf("expected Name 'John', got '%s'", bound.Name)
	}
}

func Test_BindFormMultipart_InvalidContentType(t *testing.T) {
	router := New()

	router.POST("/upload", func(ctx *Context) error {
		var bound struct {
			Name string `form:"name"`
		}
		err := ctx.BindFormMultipart(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Not multipart
	body := "name=test"
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindForm_BindingGlobalForm(t *testing.T) {
	router := New()

	type Form struct {
		ID int `form:"id"`
	}

	var bound Form
	router.POST("/form", func(ctx *Context) error {
		err := BindingForm.Bind(ctx, &bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "id=42"
	req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != 42 {
		t.Errorf("expected ID 42, got %d", bound.ID)
	}
}

func Test_BindFormPost_BindingGlobalFormPost(t *testing.T) {
	router := New()

	type Form struct {
		Name string `form:"name"`
	}

	var bound Form
	router.POST("/form", func(ctx *Context) error {
		err := BindingFormPost.Bind(ctx, &bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	body := "name=test"
	req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "test" {
		t.Errorf("expected Name 'test', got '%s'", bound.Name)
	}
}

func Test_BindFormMultipart_BindingGlobalFormMultipart(t *testing.T) {
	router := New()

	type Form struct {
		Name string `form:"name"`
	}

	var bound Form
	router.POST("/form", func(ctx *Context) error {
		err := BindingFormMultipart.Bind(ctx, &bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("name", "multipart-test")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/form", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "multipart-test" {
		t.Errorf("expected Name 'multipart-test', got '%s'", bound.Name)
	}
}
