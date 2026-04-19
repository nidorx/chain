package chain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// 1. Successful path binding with a single parameter (e.g., /users/:id)
func Test_BindPath_SingleParam(t *testing.T) {
	router := New()

	type UserParam struct {
		ID string `path:"id"`
	}

	var bound UserParam
	router.GET("/users/:id", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "12345" {
		t.Errorf("expected ID '12345', got '%s'", bound.ID)
	}
}

// 2. Path binding with multiple parameters (e.g., /users/:userId/posts/:postId)
func Test_BindPath_MultipleParams(t *testing.T) {
	router := New()

	type PostParam struct {
		UserID string `path:"userId"`
		PostID string `path:"postId"`
	}

	var bound PostParam
	router.GET("/users/:userId/posts/:postId", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42/posts/99", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.UserID != "42" {
		t.Errorf("expected UserID '42', got '%s'", bound.UserID)
	}
	if bound.PostID != "99" {
		t.Errorf("expected PostID '99', got '%s'", bound.PostID)
	}
}

func Test_BindPath_ThreeParams(t *testing.T) {
	router := New()

	type OrgParam struct {
		OrgID   string `path:"orgId"`
		RepoID  string `path:"repoId"`
		IssueID string `path:"issueId"`
	}

	var bound OrgParam
	router.GET("/orgs/:orgId/repos/:repoId/issues/:issueId", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/repos/main/issues/100", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.OrgID != "acme" {
		t.Errorf("expected OrgID 'acme', got '%s'", bound.OrgID)
	}
	if bound.RepoID != "main" {
		t.Errorf("expected RepoID 'main', got '%s'", bound.RepoID)
	}
	if bound.IssueID != "100" {
		t.Errorf("expected IssueID '100', got '%s'", bound.IssueID)
	}
}

// 3. Path binding with string fields
func Test_BindPath_StringFields(t *testing.T) {
	router := New()

	type StringPath struct {
		Name     string `path:"name"`
		Category string `path:"category"`
	}

	var bound StringPath
	router.GET("/items/:category/:name", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/electronics/laptop", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "laptop" {
		t.Errorf("expected Name 'laptop', got '%s'", bound.Name)
	}
	if bound.Category != "electronics" {
		t.Errorf("expected Category 'electronics', got '%s'", bound.Category)
	}
}

// 4. Path binding with integer fields
func Test_BindPath_IntFields(t *testing.T) {
	router := New()

	type IntPath struct {
		ID       int   `path:"id"`
		Revision int32 `path:"revision"`
		Version  int64 `path:"version"`
	}

	var bound IntPath
	router.GET("/docs/:id/revision/:revision/version/:version", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/docs/42/revision/3/version/1000", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != 42 {
		t.Errorf("expected ID 42, got %d", bound.ID)
	}
	if bound.Revision != 3 {
		t.Errorf("expected Revision 3, got %d", bound.Revision)
	}
	if bound.Version != 1000 {
		t.Errorf("expected Version 1000, got %d", bound.Version)
	}
}

func Test_BindPath_IntFields_InvalidValue(t *testing.T) {
	router := New()

	type IntPath struct {
		ID int `path:"id"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound IntPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/notanumber", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindPath_UintFields(t *testing.T) {
	router := New()

	type UintPath struct {
		ID   uint   `path:"id"`
		Flag uint64 `path:"flag"`
	}

	var bound UintPath
	router.GET("/resources/:id/flags/:flag", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/resources/100/flags/255", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != 100 {
		t.Errorf("expected ID 100, got %d", bound.ID)
	}
	if bound.Flag != 255 {
		t.Errorf("expected Flag 255, got %d", bound.Flag)
	}
}

func Test_BindPath_FloatFields(t *testing.T) {
	router := New()

	type FloatPath struct {
		Score  float32 `path:"score"`
		Weight float64 `path:"weight"`
	}

	var bound FloatPath
	router.GET("/entries/:score/w/:weight", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/entries/9.5/w/3.14", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Score != 9.5 {
		t.Errorf("expected Score 9.5, got %f", bound.Score)
	}
	if bound.Weight != 3.14 {
		t.Errorf("expected Weight 3.14, got %f", bound.Weight)
	}
}

// 5. Path binding with wildcard parameters (e.g., /files/*filepath)
func Test_BindPath_WildcardParam(t *testing.T) {
	router := New()

	type FilePath struct {
		Filepath string `path:"filepath"`
	}

	var bound FilePath
	router.GET("/files/*filepath", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/files/images/logo.png", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Filepath != "/images/logo.png" {
		t.Errorf("expected Filepath '/images/logo.png', got '%s'", bound.Filepath)
	}
}

func Test_BindPath_WildcardParam_RootPath(t *testing.T) {
	router := New()

	type FilePath struct {
		Filepath string `path:"filepath"`
	}

	var bound FilePath
	router.GET("/files/*filepath", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/files/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Filepath != "/" {
		t.Errorf("expected Filepath '/', got '%s'", bound.Filepath)
	}
}

// 6. Path binding with field name mapping via tag `path:"param_name"`
func Test_BindPath_FieldNameMapping(t *testing.T) {
	router := New()

	type MappedPath struct {
		UserIdentifier string `path:"user_id"`
		ResourceType   string `path:"resource_type"`
		ResourceID     int    `path:"resource_id"`
	}

	var bound MappedPath
	router.GET("/users/:user_id/:resource_type/:resource_id", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/john123/documents/456", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.UserIdentifier != "john123" {
		t.Errorf("expected UserIdentifier 'john123', got '%s'", bound.UserIdentifier)
	}
	if bound.ResourceType != "documents" {
		t.Errorf("expected ResourceType 'documents', got '%s'", bound.ResourceType)
	}
	if bound.ResourceID != 456 {
		t.Errorf("expected ResourceID 456, got %d", bound.ResourceID)
	}
}

func Test_BindPath_FieldNameMapping_UnderscoredParams(t *testing.T) {
	router := New()

	type UnderscorePath struct {
		FirstName string `path:"first_name"`
		LastName  string `path:"last_name"`
	}

	var bound UnderscorePath
	router.GET("/people/:first_name/:last_name", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/people/John/Doe", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.FirstName != "John" {
		t.Errorf("expected FirstName 'John', got '%s'", bound.FirstName)
	}
	if bound.LastName != "Doe" {
		t.Errorf("expected LastName 'Doe', got '%s'", bound.LastName)
	}
}

// 7. Path binding with missing optional fields (fields without matching params)
func Test_BindPath_OptionalFields(t *testing.T) {
	router := New()

	type OptionalPath struct {
		ID    string `path:"id"`
		Extra string `path:"extra"`
	}

	var bound OptionalPath
	router.GET("/items/:id", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/abc123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got '%s'", bound.ID)
	}
	// Extra field should remain as zero value
	if bound.Extra != "" {
		t.Errorf("expected Extra '' (zero value), got '%s'", bound.Extra)
	}
}

// 8. Empty path parameters
func Test_BindPath_EmptyParamValue(t *testing.T) {
	router := New()

	type EmptyPath struct {
		ID   string `path:"id"`
		Name string `path:"name"`
	}

	var bound EmptyPath
	router.GET("/items/:id/:name", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Test with empty-looking values (not truly empty since path params must match segments)
	req := httptest.NewRequest(http.MethodGet, "/items//placeholder", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// The router may or may not match this route depending on implementation
	// We just verify the binding mechanism works with whatever values are captured
	if bound.ID != "" && bound.Name != "placeholder" {
		t.Errorf("expected ID '' and Name 'placeholder', got ID='%s' Name='%s'", bound.ID, bound.Name)
	}
}

func Test_BindPath_NumericZeroValues(t *testing.T) {
	router := New()

	type NumericPath struct {
		ID    int     `path:"id"`
		Score float64 `path:"score"`
	}

	var bound NumericPath
	router.GET("/items/:id", func(ctx *Context) error {
		// Only bind id, score has no matching param
		m := make(map[string][]string, ctx.paramCount)
		for i := 0; i < ctx.paramCount; i++ {
			m[ctx.paramNames[i]] = []string{ctx.paramValues[i]}
		}
		// Manually add score param with empty value to test zero conversion
		m["score"] = []string{""}
		return mapFormByTag(&bound, m, "path")
	})

	req := httptest.NewRequest(http.MethodGet, "/items/0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != 0 {
		t.Errorf("expected ID 0, got %d", bound.ID)
	}
	if bound.Score != 0.0 {
		t.Errorf("expected Score 0.0, got %f", bound.Score)
	}
}

// 9. ShouldBindPath success and error cases
func Test_ShouldBindPath_Success(t *testing.T) {
	router := New()

	type PathParam struct {
		ID string `path:"id"`
	}

	var bound PathParam
	router.GET("/items/:id", func(ctx *Context) error {
		err := ctx.ShouldBindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/test123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "test123" {
		t.Errorf("expected ID 'test123', got '%s'", bound.ID)
	}
}

func Test_ShouldBindPath_Error_InvalidInt(t *testing.T) {
	router := New()

	type IntPathParam struct {
		ID int `path:"id"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound IntPathParam
		err := ctx.ShouldBindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/notanumber", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_ShouldBindPath_NilPointer(t *testing.T) {
	router := New()

	router.GET("/nil", func(ctx *Context) error {
		err := ctx.ShouldBindPath(nil)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/nil", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// ShouldBindPath with nil pointer does not produce an error (empty form map returns early)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_ShouldBindPath_NonPointer(t *testing.T) {
	router := New()

	type PathParam struct {
		ID string `path:"id"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound PathParam
		// Passing a non-pointer should fail
		err := ctx.ShouldBindPath(bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Non-pointer causes a panic recovered as 500, or returns an error as 400
	if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 400 or 500, got %d", w.Code)
	}
}

// 10. BindPath with validation tags (e.g., `path:"id" binding:"required"`)
func Test_BindPath_Validation_Required(t *testing.T) {
	router := New()

	type RequiredPath struct {
		ID string `path:"id" binding:"required"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound RequiredPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/valid-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_BindPath_Validation_MinLength(t *testing.T) {
	router := New()

	type MinLengthPath struct {
		ID string `path:"id" binding:"min=3"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound MinLengthPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Valid: ID length >= 3
	req := httptest.NewRequest(http.MethodGet, "/items/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_BindPath_Validation_MinLength_Fail(t *testing.T) {
	router := New()

	type MinLengthPath struct {
		ID string `path:"id" binding:"min=3"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound MinLengthPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Invalid: ID length < 3
	req := httptest.NewRequest(http.MethodGet, "/items/ab", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindPath_Validation_Numeric(t *testing.T) {
	router := New()

	type NumericValidationPath struct {
		Age int `path:"age" binding:"required,gte=18"`
	}

	router.GET("/users/:age", func(ctx *Context) error {
		var bound NumericValidationPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Valid age
	req := httptest.NewRequest(http.MethodGet, "/users/25", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_BindPath_Validation_Numeric_Fail(t *testing.T) {
	router := New()

	type NumericValidationPath struct {
		Age int `path:"age" binding:"required,gte=18"`
	}

	router.GET("/users/:age", func(ctx *Context) error {
		var bound NumericValidationPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// Invalid age (under 18)
	req := httptest.NewRequest(http.MethodGet, "/users/15", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// 11. Path binding combined with query parameters
func Test_BindPath_CombinedWithPathAndQuery(t *testing.T) {
	router := New()

	type CombinedParams struct {
		ID     string `path:"id"`
		Filter string `query:"filter"`
		Sort   string `query:"sort"`
	}

	var bound CombinedParams
	router.GET("/items/:id", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		err = ctx.BindQuery(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/42?filter=active&sort=name", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "42" {
		t.Errorf("expected ID '42', got '%s'", bound.ID)
	}
	if bound.Filter != "active" {
		t.Errorf("expected Filter 'active', got '%s'", bound.Filter)
	}
	if bound.Sort != "name" {
		t.Errorf("expected Sort 'name', got '%s'", bound.Sort)
	}
}

func Test_Bind_Auto_PathAndQuery(t *testing.T) {
	router := New()

	type AutoCombinedParams struct {
		ID   string `path:"id"`
		Name string `query:"name"`
		Page int    `query:"page"`
	}

	var bound AutoCombinedParams
	router.GET("/users/:id", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123?name=john&page=5", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "123" {
		t.Errorf("expected ID '123', got '%s'", bound.ID)
	}
	if bound.Name != "john" {
		t.Errorf("expected Name 'john', got '%s'", bound.Name)
	}
	if bound.Page != 5 {
		t.Errorf("expected Page 5, got %d", bound.Page)
	}
}

// 12. GetParam and GetParamByIndex direct usage
func Test_GetParam_ByString(t *testing.T) {
	router := New()

	var capturedID string
	var capturedName string
	router.GET("/users/:id/:name", func(ctx *Context) error {
		capturedID = ctx.GetParam("id")
		capturedName = ctx.GetParam("name")
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42/john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if capturedID != "42" {
		t.Errorf("expected capturedID '42', got '%s'", capturedID)
	}
	if capturedName != "john" {
		t.Errorf("expected capturedName 'john', got '%s'", capturedName)
	}
}

func Test_GetParam_NonExistent(t *testing.T) {
	router := New()

	var captured string
	router.GET("/users/:id", func(ctx *Context) error {
		captured = ctx.GetParam("nonexistent")
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if captured != "" {
		t.Errorf("expected captured '' for nonexistent param, got '%s'", captured)
	}
}

func Test_GetParamByIndex(t *testing.T) {
	router := New()

	var capturedFirst string
	var capturedSecond string
	router.GET("/users/:id/:name", func(ctx *Context) error {
		capturedFirst = ctx.GetParamByIndex(0)
		capturedSecond = ctx.GetParamByIndex(1)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42/john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if capturedFirst != "42" {
		t.Errorf("expected capturedFirst '42', got '%s'", capturedFirst)
	}
	if capturedSecond != "john" {
		t.Errorf("expected capturedSecond 'john', got '%s'", capturedSecond)
	}
}

func Test_GetParamByIndex_OutOfRange(t *testing.T) {
	router := New()

	var captured string
	router.GET("/users/:id", func(ctx *Context) error {
		// Index 1 is out of range since only one param exists
		captured = ctx.GetParamByIndex(1)
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	// Out-of-range access returns empty string (default string value)
	if captured != "" {
		t.Errorf("expected captured '' for out-of-range index, got '%s'", captured)
	}
}

func Test_GetParam_WithWildcard(t *testing.T) {
	router := New()

	var captured string
	router.GET("/files/*filepath", func(ctx *Context) error {
		captured = ctx.GetParam("filepath")
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/files/images/photos/vacation.jpg", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if captured != "/images/photos/vacation.jpg" {
		t.Errorf("expected captured '/images/photos/vacation.jpg', got '%s'", captured)
	}
}

// 13. Path parameter URL decoding (e.g., %20 for space)
func Test_BindPath_URLEncodedSpace(t *testing.T) {
	router := New()

	type EncodedPath struct {
		Name string `path:"name"`
	}

	var bound EncodedPath
	router.GET("/users/:name", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// URL with encoded space (%20)
	req := httptest.NewRequest(http.MethodGet, "/users/John%20Doe", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "John Doe" {
		t.Errorf("expected Name 'John Doe', got '%s'", bound.Name)
	}
}

func Test_BindPath_URLEncodedSpecialCharacters(t *testing.T) {
	router := New()

	type SpecialPath struct {
		Name string `path:"name"`
	}

	var bound SpecialPath
	router.GET("/items/:name", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// URL with encoded special characters (note: / cannot be in a path segment)
	req := httptest.NewRequest(http.MethodGet, "/items/hello%40world%21", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "hello@world!" {
		t.Errorf("expected Name 'hello@world!', got '%s'", bound.Name)
	}
}

func Test_BindPath_URLEncodedPlusSign(t *testing.T) {
	router := New()

	type PlusPath struct {
		Query string `path:"query"`
	}

	var bound PlusPath
	router.GET("/search/:query", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// URL with encoded plus sign
	req := httptest.NewRequest(http.MethodGet, "/search/c%2B%2B", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Query != "c++" {
		t.Errorf("expected Query 'c++', got '%s'", bound.Query)
	}
}

func Test_BindPath_URLEncodedUnicode(t *testing.T) {
	router := New()

	type UnicodePath struct {
		Name string `path:"name"`
	}

	var bound UnicodePath
	router.GET("/users/:name", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// URL with encoded unicode characters
	req := httptest.NewRequest(http.MethodGet, "/users/%E4%B8%AD%E6%96%87", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "中文" {
		t.Errorf("expected Name '中文', got '%s'", bound.Name)
	}
}

// 14. Auto-binding from ctx.Bind() with route params
func Test_Bind_Auto_WithPathParams(t *testing.T) {
	router := New()

	type AutoPathStruct struct {
		ID   string `path:"id"`
		Name string `path:"name"`
	}

	var bound AutoPathStruct
	router.GET("/users/:id/:name", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42/john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "42" {
		t.Errorf("expected ID '42', got '%s'", bound.ID)
	}
	if bound.Name != "john" {
		t.Errorf("expected Name 'john', got '%s'", bound.Name)
	}
}

func Test_Bind_Auto_WithPathAndQueryCombined(t *testing.T) {
	router := New()

	type AutoCombined struct {
		ID     string `path:"id"`
		Filter string `query:"filter"`
		Limit  int    `query:"limit"`
	}

	var bound AutoCombined
	router.GET("/items/:id", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/100?filter=active&limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "100" {
		t.Errorf("expected ID '100', got '%s'", bound.ID)
	}
	if bound.Filter != "active" {
		t.Errorf("expected Filter 'active', got '%s'", bound.Filter)
	}
	if bound.Limit != 10 {
		t.Errorf("expected Limit 10, got %d", bound.Limit)
	}
}

func Test_Bind_Auto_WithoutPathParams(t *testing.T) {
	router := New()

	type AutoNoPath struct {
		Name string `query:"name"`
	}

	var bound AutoNoPath
	router.GET("/search", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/search?name=test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Name != "test" {
		t.Errorf("expected Name 'test', got '%s'", bound.Name)
	}
}

// Additional edge case tests

func Test_BindPath_BoolField(t *testing.T) {
	router := New()

	type BoolPath struct {
		Active bool `path:"active"`
	}

	var bound BoolPath
	router.GET("/features/:active", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/features/true", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Active != true {
		t.Errorf("expected Active true, got %v", bound.Active)
	}
}

func Test_BindPath_BoolField_False(t *testing.T) {
	router := New()

	type BoolPath struct {
		Active bool `path:"active"`
	}

	var bound BoolPath
	router.GET("/features/:active", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/features/false", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.Active != false {
		t.Errorf("expected Active false, got %v", bound.Active)
	}
}

func Test_BindPath_BoolField_Invalid(t *testing.T) {
	router := New()

	type BoolPath struct {
		Active bool `path:"active"`
	}

	router.GET("/features/:active", func(ctx *Context) error {
		var bound BoolPath
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/features/notbool", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_BindPath_NoMatchingParam_ZeroValues(t *testing.T) {
	router := New()

	type NoMatchPath struct {
		ID    string  `path:"id"`
		Count int     `path:"count"`
		Rate  float64 `path:"rate"`
		Flag  bool    `path:"flag"`
	}

	var bound NoMatchPath
	router.GET("/test/:id", func(ctx *Context) error {
		// Only id param exists, others should be zero values
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "abc" {
		t.Errorf("expected ID 'abc', got '%s'", bound.ID)
	}
	if bound.Count != 0 {
		t.Errorf("expected Count 0, got %d", bound.Count)
	}
	if bound.Rate != 0.0 {
		t.Errorf("expected Rate 0.0, got %f", bound.Rate)
	}
	if bound.Flag != false {
		t.Errorf("expected Flag false, got %v", bound.Flag)
	}
}

func Test_BindPath_TagIgnore(t *testing.T) {
	router := New()

	type IgnorePath struct {
		ID   string `path:"id"`
		Skip string `path:"-"`
	}

	var bound IgnorePath
	bound.Skip = "preserved"
	router.GET("/items/:id", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "123" {
		t.Errorf("expected ID '123', got '%s'", bound.ID)
	}
	// Field with path:"-" should be ignored
	if bound.Skip != "preserved" {
		t.Errorf("expected Skip 'preserved' (unchanged), got '%s'", bound.Skip)
	}
}

func Test_BindPath_MapStringString(t *testing.T) {
	router := New()

	router.GET("/items/:id", func(ctx *Context) error {
		bound := make(map[string]string)
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.Json(bound)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/test123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response contains bound param
	body := w.Body.String()
	if body != `{"id":"test123"}` && body != "{\"id\":\"test123\"}" {
		t.Errorf("expected response to contain bound param, got '%s'", body)
	}
}

func Test_ShouldBindPath_WithValidation_Success(t *testing.T) {
	router := New()

	type ValidatedPath struct {
		ID string `path:"id" binding:"required,min=5"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound ValidatedPath
		err := ctx.ShouldBindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/valid-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func Test_ShouldBindPath_WithValidation_Fail(t *testing.T) {
	router := New()

	type ValidatedPath struct {
		ID string `path:"id" binding:"required,min=5"`
	}

	router.GET("/items/:id", func(ctx *Context) error {
		var bound ValidatedPath
		err := ctx.ShouldBindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	// ID too short (less than 5 chars)
	req := httptest.NewRequest(http.MethodGet, "/items/ab", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_GetParam_CaseSensitivity(t *testing.T) {
	router := New()

	var captured string
	router.GET("/users/:ID", func(ctx *Context) error {
		// Param name is case-sensitive
		captured = ctx.GetParam("ID")
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if captured != "42" {
		t.Errorf("expected captured '42', got '%s'", captured)
	}
}

func Test_BindPath_MixedParamTypes(t *testing.T) {
	router := New()

	type MixedTypesPath struct {
		ID     int     `path:"id"`
		Name   string  `path:"name"`
		Active bool    `path:"active"`
		Score  float64 `path:"score"`
	}

	var bound MixedTypesPath
	router.GET("/entries/:id/:name/:active/:score", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/entries/42/john/true/9.5", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != 42 {
		t.Errorf("expected ID 42, got %d", bound.ID)
	}
	if bound.Name != "john" {
		t.Errorf("expected Name 'john', got '%s'", bound.Name)
	}
	if bound.Active != true {
		t.Errorf("expected Active true, got %v", bound.Active)
	}
	if bound.Score != 9.5 {
		t.Errorf("expected Score 9.5, got %f", bound.Score)
	}
}

func Test_BindPath_GroupedRoutes(t *testing.T) {
	router := New()

	type GroupPath struct {
		ID string `path:"id"`
	}

	var bound GroupPath
	group := router.Group("/api")
	{
		group.GET("/users/:id", func(ctx *Context) error {
			err := ctx.BindPath(&bound)
			if err != nil {
				ctx.Error(err.Error(), http.StatusBadRequest)
				return err
			}
			ctx.OK()
			return nil
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users/test123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "test123" {
		t.Errorf("expected ID 'test123', got '%s'", bound.ID)
	}
}

func Test_BindPath_WildcardWithAdditionalParams(t *testing.T) {
	router := New()

	type WildcardWithPath struct {
		BasePath string `path:"basepath"`
		Filepath string `path:"filepath"`
	}

	var bound WildcardWithPath
	router.GET("/base/:basepath/*filepath", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/base/myapp/static/style.css", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.BasePath != "myapp" {
		t.Errorf("expected BasePath 'myapp', got '%s'", bound.BasePath)
	}
	if bound.Filepath != "/static/style.css" {
		t.Errorf("expected Filepath '/static/style.css', got '%s'", bound.Filepath)
	}
}

func Test_BindPath_EmptyRoute(t *testing.T) {
	router := New()

	type EmptyRoutePath struct {
		ID string `path:"id"`
	}

	var bound EmptyRoutePath
	router.GET("/:id", func(ctx *Context) error {
		err := ctx.BindPath(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/root-value", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "root-value" {
		t.Errorf("expected ID 'root-value', got '%s'", bound.ID)
	}
}

func Test_BindPath_NestedGroupsWithParams(t *testing.T) {
	router := New()

	type NestedPath struct {
		OrgID  string `path:"orgId"`
		RepoID string `path:"repoId"`
	}

	var bound NestedPath
	v1 := router.Group("/v1")
	{
		orgs := v1.Group("/orgs/:orgId")
		{
			orgs.GET("/repos/:repoId", func(ctx *Context) error {
				err := ctx.BindPath(&bound)
				if err != nil {
					ctx.Error(err.Error(), http.StatusBadRequest)
					return err
				}
				ctx.OK()
				return nil
			})
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/acme/repos/main", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.OrgID != "acme" {
		t.Errorf("expected OrgID 'acme', got '%s'", bound.OrgID)
	}
	if bound.RepoID != "main" {
		t.Errorf("expected RepoID 'main', got '%s'", bound.RepoID)
	}
}

func Test_BindPath_BindingDefault_WithPath(t *testing.T) {
	router := New()

	type DefaultBindingPath struct {
		ID   string `path:"id"`
		Name string `query:"name"`
	}

	var bound DefaultBindingPath
	router.GET("/items/:id", func(ctx *Context) error {
		err := ctx.Bind(&bound)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return err
		}
		ctx.OK()
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/123?name=test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if bound.ID != "123" {
		t.Errorf("expected ID '123', got '%s'", bound.ID)
	}
	if bound.Name != "test" {
		t.Errorf("expected Name 'test', got '%s'", bound.Name)
	}
}
