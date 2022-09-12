// Copyright 2022 Alex Rodin. All rights reserved.
// Based on the httprouter package, Copyright 2013 Julien Schmidt.

package chain

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func Test_Router(t *testing.T) {
	router := New()

	routed := false
	router.Handle(http.MethodGet, "/user/:name", func(ctx *Context) error {
		routed = true
		want := "gopher"
		got := ctx.GetParam("name")
		if got != want {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, got)
		}
		return nil
	})

	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/user/gopher", nil)
	router.ServeHTTP(w, req)

	if !routed {
		t.Fatal("routing failed")
	}
}

type handlerStruct struct {
	handled *bool
}

func (h handlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	*h.handled = true
}

func Test_Router_API(t *testing.T) {
	var get, head, options, post, put, patch, delete, handler, handlerFunc bool

	httpHandler := handlerStruct{&handler}

	router := New()
	router.GET("/GET", func(ctx *Context) error {
		get = true
		return nil
	})
	router.HEAD("/GET", func(ctx *Context) error {
		head = true
		return nil
	})
	router.OPTIONS("/GET", func(ctx *Context) error {
		options = true
		return nil
	})
	router.POST("/POST", func(ctx *Context) error {
		post = true
		return nil
	})
	router.PUT("/PUT", func(ctx *Context) error {
		put = true
		return nil
	})
	router.PATCH("/PATCH", func(ctx *Context) error {
		patch = true
		return nil
	})
	router.DELETE("/DELETE", func(ctx *Context) error {
		delete = true
		return nil
	})
	router.Handle(http.MethodGet, "/Handle", httpHandler)
	router.Handle(http.MethodGet, "/HandlerFunc", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc = true
	})

	w := new(mockResponseWriter)

	r, _ := http.NewRequest(http.MethodGet, "/GET", nil)
	router.ServeHTTP(w, r)
	if !get {
		t.Error("routing GET failed")
	}

	r, _ = http.NewRequest(http.MethodHead, "/GET", nil)
	router.ServeHTTP(w, r)
	if !head {
		t.Error("routing HEAD failed")
	}

	r, _ = http.NewRequest(http.MethodOptions, "/GET", nil)
	router.ServeHTTP(w, r)
	if !options {
		t.Error("routing OPTIONS failed")
	}

	r, _ = http.NewRequest(http.MethodPost, "/POST", nil)
	router.ServeHTTP(w, r)
	if !post {
		t.Error("routing POST failed")
	}

	r, _ = http.NewRequest(http.MethodPut, "/PUT", nil)
	router.ServeHTTP(w, r)
	if !put {
		t.Error("routing PUT failed")
	}

	r, _ = http.NewRequest(http.MethodPatch, "/PATCH", nil)
	router.ServeHTTP(w, r)
	if !patch {
		t.Error("routing PATCH failed")
	}

	r, _ = http.NewRequest(http.MethodDelete, "/DELETE", nil)
	router.ServeHTTP(w, r)
	if !delete {
		t.Error("routing DELETE failed")
	}

	r, _ = http.NewRequest(http.MethodGet, "/Handle", nil)
	router.ServeHTTP(w, r)
	if !handler {
		t.Error("routing Handle failed")
	}

	r, _ = http.NewRequest(http.MethodGet, "/HandlerFunc", nil)
	router.ServeHTTP(w, r)
	if !handlerFunc {
		t.Error("routing HandlerFunc failed")
	}
}

func Test_Router_Invalid_Input(t *testing.T) {
	router := New()

	handle := func(ctx *Context) error {
		return nil
	}

	recv := catchPanic(func() {
		router.Handle("", "/", handle)
	})
	if recv == nil {
		t.Fatal("registering empty method did not panic")
	}

	recv = catchPanic(func() {
		router.GET("", handle)
	})
	if recv == nil {
		t.Fatal("registering empty path did not panic")
	}

	recv = catchPanic(func() {
		router.GET("noSlashRoot", handle)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}

	recv = catchPanic(func() {
		router.GET("/", nil)
	})
	if recv == nil {
		t.Fatal("registering nil handler did not panic")
	}
}

func Test_Router_Chaining(t *testing.T) {
	router1 := New()
	router2 := New()
	router1.NotFoundHandler = router2

	fooHit := false
	router1.POST("/foo", func(ctx *Context) error {
		fooHit = true
		ctx.Writer.WriteHeader(http.StatusOK)
		return nil
	})

	barHit := false
	router2.POST("/bar", func(ctx *Context) error {
		barHit = true
		ctx.Writer.WriteHeader(http.StatusOK)
		return nil
	})

	r, _ := http.NewRequest(http.MethodPost, "/foo", nil)
	w := httptest.NewRecorder()
	router1.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK && fooHit) {
		t.Errorf("Regular routing failed with router chaining.")
		t.FailNow()
	}

	r, _ = http.NewRequest(http.MethodPost, "/bar", nil)
	w = httptest.NewRecorder()
	router1.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK && barHit) {
		t.Errorf("Chained routing failed with router chaining.")
		t.FailNow()
	}

	r, _ = http.NewRequest(http.MethodPost, "/qax", nil)
	w = httptest.NewRecorder()
	router1.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("NotFound behavior failed with router chaining.")
		t.FailNow()
	}
}

func Test_Router_OPTIONS(t *testing.T) {
	handlerFunc := func(ctx *Context) error {
		return nil
	}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	// * (server)
	r, _ := http.NewRequest(http.MethodOptions, "*", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	r, _ = http.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	r, _ = http.NewRequest(http.MethodOptions, "/doesnotexist", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// add another method
	router.GET("/path", handlerFunc)

	// set a global OPTIONS handler
	router.GlobalOPTIONSHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	// test again
	// * (server)
	r, _ = http.NewRequest(http.MethodOptions, "*", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	r, _ = http.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// custom handler
	var custom bool
	router.OPTIONS("/path", func(ctx *Context) error {
		custom = true
		return nil
	})

	// test again
	// * (server)
	r, _ = http.NewRequest(http.MethodOptions, "*", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
	if custom {
		t.Error("custom handler called on *")
	}

	// path
	r, _ = http.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	}
	if !custom {
		t.Error("custom handler not called")
	}
}

func Test_Router_NotAllowed(t *testing.T) {
	handlerFunc := func(ctx *Context) error {
		return nil
	}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	r, _ := http.NewRequest(http.MethodGet, "/path", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// add another method
	router.DELETE("/path", handlerFunc)
	router.OPTIONS("/path", handlerFunc) // must be ignored

	// test again
	r, _ = http.NewRequest(http.MethodGet, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// test custom handler
	w = httptest.NewRecorder()
	responseText := "custom method"
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(responseText))
	})
	router.ServeHTTP(w, r)
	if got := w.Body.String(); !(got == responseText) {
		t.Errorf("unexpected response got %q want %q", got, responseText)
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("unexpected response code %d want %d", w.Code, http.StatusTeapot)
	}
	if allow := w.Header().Get("Allow"); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
}

func Test_Router_NotFound(t *testing.T) {
	handlerFunc := func(ctx *Context) error {
		return nil
	}

	router := New()
	router.GET("/path", handlerFunc)
	router.GET("/dir/", handlerFunc)
	router.GET("/path/:param", handlerFunc)
	router.GET("/path/:param/*", handlerFunc)
	router.GET("/", handlerFunc)

	testRoutes := []struct {
		route    string
		code     int
		location string
	}{
		{"/path/", http.StatusMovedPermanently, "/path"},                                // TSR -/
		{"/dir", http.StatusMovedPermanently, "/dir/"},                                  // TSR +/
		{"", http.StatusMovedPermanently, "/"},                                          // TSR +/
		{"/PATH", http.StatusMovedPermanently, "/path"},                                 // Fixed Case
		{"/PATH/foo", http.StatusMovedPermanently, "/path/foo"},                         // Fixed Case
		{"/PATH/foo/bar", http.StatusMovedPermanently, "/path/foo/bar"},                 // Fixed Case
		{"/PATH/foo/bar/baz", http.StatusMovedPermanently, "/path/foo/bar/baz"},         // Fixed Case
		{"/PATH/foo/bar/baz/qux", http.StatusMovedPermanently, "/path/foo/bar/baz/qux"}, // Fixed Case
		{"/DIR/", http.StatusMovedPermanently, "/dir/"},                                 // Fixed Case
		{"/PATH/", http.StatusMovedPermanently, "/path"},                                // Fixed Case -/
		{"/DIR", http.StatusMovedPermanently, "/dir/"},                                  // Fixed Case +/
		{"/../path", http.StatusMovedPermanently, "/path"},                              // CleanPath
		{"/nope", http.StatusNotFound, ""},                                              // NotFound
	}
	for _, tr := range testRoutes {
		t.Run(tr.route, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, tr.route, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tr.code {
				t.Errorf("NotFound handling route %s failed: Invalid Code\n   actual: %v\n expected: %v", tr.route, w.Code, tr.code)
			} else if fmt.Sprint(w.Header().Get("Location")) != tr.location {
				t.Errorf("NotFound handling route %s failed: Invalid Location\n   actual: %v\n expected: %v", tr.route, w.Header().Get("Location"), tr.location)
			}
			//if !(w.Code == tr.code && (w.Code == http.StatusNotFound || fmt.Sprint(w.Header().Get("Location")) == tr.location)) {
			//	t.Errorf("NotFound handling route %s failed: Code=%d, Header=%v", tr.route, w.Code, w.Header().Get("Location"))
			//}
		})
	}

	// Test custom not found handler
	var notFound bool
	router.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		notFound = true
	})
	r, _ := http.NewRequest(http.MethodGet, "/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound && notFound == true) {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// Test other method than GET (want 308 instead of 301)
	router.PATCH("/path", handlerFunc)
	r, _ = http.NewRequest(http.MethodPatch, "/path/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusPermanentRedirect && fmt.Sprint(w.Header()) == "map[Location:[/path]]") {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", handlerFunc)
	r, _ = http.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("NotFound handling route / failed: Code=%d", w.Code)
	}
}

func Test_Router_PanicHandler(t *testing.T) {
	router := New()
	panicHandled := false

	router.PanicHandler = func(rw http.ResponseWriter, r *http.Request, p interface{}) {
		panicHandled = true
	}

	router.Handle(http.MethodPut, "/user/:name", func(ctx *Context) error {
		panic(any("oops!"))
		return nil
	})

	w := new(mockResponseWriter)
	req, _ := http.NewRequest(http.MethodPut, "/user/gopher", nil)

	defer func() {
		if rcv := recover(); rcv != any(nil) {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !panicHandled {
		t.Fatal("simulating failed")
	}
}

func Test_Router_ErrorHandler(t *testing.T) {
	router := New()
	errorHandled := false

	router.ErrorHandler = func(context *Context, err error) {
		errorHandled = true
	}

	router.Handle(http.MethodPut, "/user/:name", func(ctx *Context) error {
		return errors.New("oops!")
	})

	w := new(mockResponseWriter)
	req, _ := http.NewRequest(http.MethodPut, "/user/gopher", nil)

	defer func() {
		if rcv := recover(); rcv != any(nil) {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !errorHandled {
		t.Fatal("simulating failed")
	}
}

func Test_Router_ErrorHandler_Middleware_Before_Next(t *testing.T) {
	router := New()
	routeCalled := false
	errorHandled := false

	router.ErrorHandler = func(context *Context, err error) {
		errorHandled = true
	}

	router.Handle(http.MethodPut, "/user/:name", func(ctx *Context) error {
		// do nothing
		routeCalled = true
		return nil
	})

	router.Use(http.MethodPut, "/user/*", func() error {
		// before call next
		return errors.New("oops!")
	})

	w := new(mockResponseWriter)
	req, _ := http.NewRequest(http.MethodPut, "/user/gopher", nil)

	defer func() {
		if rcv := recover(); rcv != any(nil) {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !errorHandled {
		t.Fatal("simulating failed")
	}

	if routeCalled {
		t.Fatal("simulating failed")
	}
}

func Test_Router_ErrorHandler_Middleware_After_Next(t *testing.T) {
	router := New()
	routeCalled := false
	errorHandled := false

	router.ErrorHandler = func(context *Context, err error) {
		errorHandled = true
	}

	router.Handle(http.MethodPut, "/user/:name", func(ctx *Context) error {
		// do nothing
		routeCalled = true
		return nil
	})

	router.Use(http.MethodPut, "/user/*", func(next func() error) error {
		next()
		return errors.New("oops!")
	})

	w := new(mockResponseWriter)
	req, _ := http.NewRequest(http.MethodPut, "/user/gopher", nil)

	defer func() {
		if rcv := recover(); rcv != any(nil) {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !errorHandled {
		t.Fatal("simulating failed")
	}

	if !routeCalled {
		t.Fatal("simulating failed")
	}
}

func Test_Router_Lookup(t *testing.T) {
	routed := false
	wantHandle := func(ctx *Context) error {
		routed = true
		return nil
	}
	wantParam := "gopher"

	router := New()

	// try empty router first
	handle, _ := router.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}

	// insert route and try again
	router.GET("/user/:name", wantHandle)
	handle, ctx := router.Lookup(http.MethodGet, "/user/gopher")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle.Dispatch(ctx)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}
	got := ctx.GetParamByIndex(0)
	if got != wantParam {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParam, got)
	}
	routed = false

	// route without param
	router.GET("/user", wantHandle)
	handle, ctx = router.Lookup(http.MethodGet, "/user")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle.Dispatch(ctx)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}
	if ctx.GetParam("name") != "" {
		t.Fatalf("Wrong parameter values: want %v, got %v", nil, ctx.GetParam("name"))
	}

	handle, ctx = router.Lookup(http.MethodGet, "/user/gopher/")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}

	handle, ctx = router.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
}

func Test_Router_Params_From_Context(t *testing.T) {
	routed := false

	wantParams := "gopher"
	handlerFunc := func(_ http.ResponseWriter, req *http.Request) {
		// get params from request context
		ctx := GetContext(req.Context())
		if ctx.GetParam("name") != wantParams {
			t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, ctx.GetParam("name"))
		}
		routed = true
	}

	handlerFuncNil := func(_ http.ResponseWriter, req *http.Request) {
		// get params from request context
		ctx := GetContext(req.Context())
		if ctx.GetParam("name") != "" {
			t.Fatalf("Wrong parameter values: want %v, got %v", nil, ctx.GetParam("name"))
		}
		routed = true
	}
	router := New()
	router.Handle(http.MethodGet, "/user", handlerFuncNil)
	router.Handle(http.MethodGet, "/user/:name", handlerFunc)

	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/user/gopher", nil)
	router.ServeHTTP(w, r)
	if !routed {
		t.Fatal("Routing failed!")
	}

	routed = false
	r, _ = http.NewRequest(http.MethodGet, "/user", nil)
	router.ServeHTTP(w, r)
	if !routed {
		t.Fatal("Routing failed!")
	}
}

func Test_Router_Matched_Route_Path(t *testing.T) {
	route1 := "/user/:name"
	routed1 := false
	handle1 := func(ctx *Context) error {
		route := ctx.MatchedRoutePath
		if route != route1 {
			t.Fatalf("Wrong matched route: want %s, got %s", route1, route)
		}
		routed1 = true
		return nil
	}

	route2 := "/user/:name/details"
	routed2 := false
	handle2 := func(ctx *Context) error {
		route := ctx.MatchedRoutePath
		if route != route2 {
			t.Fatalf("Wrong matched route: want %s, got %s", route2, route)
		}
		routed2 = true
		return nil
	}

	route3 := "/"
	routed3 := false
	handle3 := func(ctx *Context) error {
		route := ctx.MatchedRoutePath
		if route != route3 {
			t.Fatalf("Wrong matched route: want %s, got %s", route3, route)
		}
		routed3 = true
		return nil
	}

	router := New()
	router.Handle(http.MethodGet, route1, handle1)
	router.Handle(http.MethodGet, route2, handle2)
	router.Handle(http.MethodGet, route3, handle3)

	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/user/gopher", nil)
	router.ServeHTTP(w, r)
	if !routed1 || routed2 || routed3 {
		t.Fatal("Routing failed!")
	}

	w = new(mockResponseWriter)
	r, _ = http.NewRequest(http.MethodGet, "/user/gopher/details", nil)
	router.ServeHTTP(w, r)
	if !routed2 || routed3 {
		t.Fatal("Routing failed!")
	}

	w = new(mockResponseWriter)
	r, _ = http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, r)
	if !routed3 {
		t.Fatal("Routing failed!")
	}
}

func Test_Router_Path_Priority(t *testing.T) {
	router := New()

	// the more specific the path, the higher its priority
	routes := []string{
		"/blog/category/page/subpage",
		"/blog/category/page/:subpage",
		"/blog/category/page/*subpage",
		"/blog/category/:page/:subpage",
		"/blog/category/:page/*subpage",
		"/blog/category/:page/subpage",
		"/blog/:category/page/subpage",
		"/blog/:category/page/:subpage",
		"/blog/:category/page/*subpage",
		"/blog/:category/:page/subpage",
		"/blog/:category/:page/:subpage",
		"/blog/:category/:page/*subpage",
		"/:blog/category/page/subpage",
		"/:blog/category/page/:subpage",
		"/:blog/category/page/*subpage",
		"/:blog/category/:page/subpage",
		"/:blog/category/:page/:subpage",
		"/:blog/category/:page/*subpage",
		"/:blog/:category/page/subpage",
		"/:blog/:category/page/:subpage",
		"/:blog/:category/page/*subpage",
		"/:blog/:category/:page/subpage",
		"/:blog/:category/:page/:subpage",
		"/:blog/:category/:page/*subpage",
		"/blog/category/page",
		"/blog/category/:page",
		"/blog/:category/page",
		"/blog/:category/:page",
		"/:blog/category/page",
		"/:blog/category/:page",
		"/:blog/:category/page",
		"/:blog/:category/:page",
		"/blog/category",
		"/blog/:category",
		"/blog",
		"/:blog",
		"/*blog",
		"/",
	}
	for _, route := range routes {
		router.GET(route, fakeHandler(route))
	}

	for _, route := range routes {
		parts := strings.Split(route, "/")
		path := ""
		var request = tRequest{
			params: map[string]string{},
		}
		for i, part := range parts {
			if i == 0 {
				continue
			}
			path = path + "/"
			if strings.HasPrefix(part, ":") {
				path = path + "param_" + strconv.Itoa(i)
				request.params[part[1:]] = "param_" + strconv.Itoa(i)
			} else if strings.HasPrefix(part, "*") {
				path = path + "wildcard_" + strconv.Itoa(i) + "/x"
				request.params[part[1:]] = "/wildcard_" + strconv.Itoa(i) + "/x"
			} else {
				path = path + part
			}
		}
		request.path = path
		request.route = route
		t.Run(route, func(t *testing.T) {
			checkRequests(t, router, request)
		})
	}
}

func Test_Router_Add_and_Get(t *testing.T) {
	router := New()

	routes := [...]string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}
	for _, route := range routes {
		router.GET(route, fakeHandler(route))
	}

	requests := []tRequest{
		{"/a", "/a", false, nil},
		{"/", "", true, nil},
		{"/hi", "/hi", false, nil},
		{"/contact", "/contact", false, nil},
		{"/co", "/co", false, nil},
		{"/con", "", true, nil},  // key mismatch
		{"/cona", "", true, nil}, // key mismatch
		{"/no", "", true, nil},   // no matching child
		{"/ab", "/ab", false, nil},
		{"/α", "/α", false, nil},
		{"/β", "/β", false, nil},
	}
	for _, tt := range requests {
		t.Run(tt.path, func(t *testing.T) {
			checkRequests(t, router, tt)
		})
	}
}

func Test_Wildcard(t *testing.T) {
	router := New()

	routes := [...]string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool",
		"/src/*filepath",
		"/src/js/:folder/:name/:file",
		"/src/:type/vendors/:name/index",
		"/src/css/:folder/:name/:file",
		"/src/:type/c/:name/index",
		"/search/",
		"/search/:query",
		"/user/:name",
		"/user/:name/about",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/info/:user/public",
		"/info/:user/project/:project",
		"/src/a/:folder/:name/:file",
		"/src/:type/b/:name/index",
		"/src/b/:folder/:name/:file",
		"/src/c/:folder/:name/:file",
		"/src/:type/a/:name/index",
		"/src/d/:folder/:name/*file",
	}
	for _, route := range routes {
		router.GET(route, fakeHandler(route))
	}

	requests := []tRequest{
		{"/cmd/value", "/cmd/:tool", false, map[string]string{"tool": "value"}},
		{"/cmd/value1/value2", "/cmd/:tool/:sub", false, map[string]string{"tool": "value1", "sub": "value2"}},
		//{"/", "/", nil},
		{"/cmd/test", "/cmd/:tool", false, map[string]string{"tool": "test"}},
		{"/cmd/test/3", "/cmd/:tool/:sub", false, map[string]string{"tool": "test", "sub": "3"}},
		{"/src/", "/src/*filepath", false, map[string]string{"filepath": "/"}},
		{"/src/some/file.png", "/src/*filepath", false, map[string]string{"filepath": "/some/file.png"}},
		{"/search/", "/search/", false, nil}, // map[string]string{"tool": "test"}
		{"/search/someth!ng+in+ünìcodé", "/search/:query", false, map[string]string{"query": "someth!ng+in+ünìcodé"}},
		{"/user/gopher", "/user/:name", false, map[string]string{"name": "gopher"}},
		{"/user/gopher/about", "/user/:name/about", false, map[string]string{"name": "gopher"}},
		{"/files/js/inc/framework.js", "/files/:dir/*filepath", false, map[string]string{"dir": "js", "filepath": "/inc/framework.js"}},
		{"/info/gordon/public", "/info/:user/public", false, map[string]string{"user": "gordon"}},
		{"/info/gordon/project/go", "/info/:user/project/:project", false, map[string]string{"user": "gordon", "project": "go"}},
		{"/src/js/vendors/jquery/main.js", "/src/js/:folder/:name/:file", false,
			map[string]string{"folder": "vendors", "name": "jquery", "file": "main.js"},
		},
		{"/src/css/vendors/jquery/main.css", "/src/css/:folder/:name/:file", false,
			map[string]string{"folder": "vendors", "name": "jquery", "file": "main.css"},
		},
		{"/src/tpl/vendors/jquery/index", "/src/:type/vendors/:name/index", false,
			map[string]string{"type": "tpl", "name": "jquery"},
		},
	}
	for _, tt := range requests {
		t.Run(tt.path, func(t *testing.T) {
			checkRequests(t, router, tt)
		})
	}
}

func Test_Wildcard_Conflict(t *testing.T) {
	router := New()

	routes := []tRoute{
		{"/src/*filepath", false},
		{"/src/*filepathx", true},
		{"/src/*", true},
		{"/src/", false},
		{"/src1/", false},
		{"/src1/*filepath", false},
		{"/src2/*filepath", false},
		{"/search/:query", false},
		{"/search/invalid", false},
		{"/user/:name", false},
		{"/user/x", false},
		{"/user/:id", true},
		{"/id/:id", false},
		{"/id/:uuid", true},
		{"/user/:id/:action", false},
		{"/user/:id/update", false},
	}
	for _, tt := range routes {
		t.Run(tt.path, func(t *testing.T) {
			recv := catchPanic(func() {
				router.GET(tt.path, fakeHandler(tt.path))
			})
			if tt.conflict {
				if recv == nil {
					t.Errorf("no panic for conflicting Route '%s'", tt.path)
				}
			} else if recv != nil {
				t.Errorf("unexpected panic for Route '%s': %v", tt.path, recv)
			}
		})
	}
}

func Test_Duplicate_Path(t *testing.T) {
	router := New()

	routes := [...]string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user/:name",
	}
	for _, route := range routes {
		recv := catchPanic(func() {
			router.GET(route, fakeHandler(route))
		})
		if recv != nil {
			t.Fatalf("panic inserting Route '%s': %v", route, recv)
		}

		// Add again
		recv = catchPanic(func() {
			router.GET(route, nil)
		})
		if recv == nil {
			t.Fatalf("no panic while inserting duplicate Route '%s", route)
		}
	}

	requests := []tRequest{
		{"/", "/", false, nil},
		{"/doc/", "/doc/", false, nil},
		{"/src/some/file.png", "/src/*filepath", false, map[string]string{"filepath": "/some/file.png"}},
		{"/search/someth!ng+in+ünìcodé", "/search/:query", false, map[string]string{"query": "someth!ng+in+ünìcodé"}},
		{"/user/gopher", "/user/:name", false, map[string]string{"name": "gopher"}},
	}
	for _, tt := range requests {
		t.Run(tt.path, func(t *testing.T) {
			checkRequests(t, router, tt)
		})
	}
}

func Test_Empty_Wildcard_Name(t *testing.T) {
	router := New()

	routes := [...]struct {
		input string
	}{
		{"/user:"},
		{"/user:/"},
		{"/cmd/:/"},
	}
	for _, tt := range routes {
		t.Run(tt.input, func(t *testing.T) {
			recv := catchPanic(func() {
				router.GET(tt.input, nil)
			})
			if recv == nil {
				t.Fatalf("no panic while inserting Route with empty wildcard name '%s", tt.input)
			}
		})
	}
}

func Test_CatchAll_Conflict(t *testing.T) {
	routes := []struct {
		first  string
		second string
	}{
		{"/src/*filepath", "/src/*"},
		{"/*", "/*filepath"},
	}
	for _, tt := range routes {
		t.Run(tt.first, func(t *testing.T) {
			router := New()
			router.GET(tt.first, fakeHandler(tt.first))
			recv := catchPanic(func() {
				router.GET(tt.second, nil)
			})

			if recv == nil {
				t.Errorf("no panic for conflicting Route '%s'", tt.first)
			}
		})
	}
}

func Test_Catch_Max_Params(t *testing.T) {
	router := New()
	var route = "/cmd/*filepath"
	router.GET(route, fakeHandler(route))
}

func Test_Double_Wildcard(t *testing.T) {
	const panicMsg = "only one wildcard per path segment is allowed in"

	routes := []struct {
		path string
	}{
		{"/:foo:bar"},
		{"/:foo:bar/"},
		{"/:foo*bar"},
	}
	for _, tt := range routes {
		t.Run(tt.path, func(t *testing.T) {
			router := New()
			recv := catchPanic(func() {
				router.GET(tt.path, fakeHandler(tt.path))
			})

			rs := fmt.Sprintf("%v", recv)
			if !strings.HasPrefix(rs, panicMsg) {
				t.Fatalf(`"Expected panic "%s" for Route '%s', got "%v"`, panicMsg, tt, recv)
			}
		})
	}
}

// Used as a workaround since we can't compare functions or their addresses
var fakeHandlerValue string

func fakeHandler(val string) Handle {
	return func(impl *Context) error {
		fakeHandlerValue = val
		return nil
	}
}

type tRequest struct {
	path       string
	route      string
	nilHandler bool
	params     map[string]string
}

type tRoute struct {
	path     string
	conflict bool
}

func checkRequests(t *testing.T, router *Router, request tRequest) {
	route, ctx := router.Lookup(http.MethodGet, request.path)
	if route == nil {
		if !request.nilHandler {
			t.Errorf("handle mismatch for path '%s'\n   actual: nil\n expected: %v", request.path, request.route)
		}
	} else if request.nilHandler {
		t.Errorf("handle mismatch for path '%s'\n   actual: %v\n expected: nil", request.path, request.route)
	} else {
		route.Dispatch(ctx)
		if fakeHandlerValue != request.route {
			//t.Errorf("getPathInfo(string) | invalid 'hasStatic'\n   actual: %v\n expected: %v", a.hasStatic, e.hasStatic)
			t.Errorf("handle mismatch for path '%s'\n   actual: %v\n expected: %v", request.path, fakeHandlerValue, request.route)
		} else if request.params != nil {
			for key, value := range request.params {
				pvalue := ctx.GetParam(key)
				if value != pvalue {
					t.Errorf("router.Lookup() | invalid 'param'\n     path: %v\n    param: %v\n   actual: %v\n expected: %v", request.path, key, pvalue, value)
				}
			}
		}
	}

}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}
