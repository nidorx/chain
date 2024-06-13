// Copyright 2022 Alex Rodin. All rights reserved.
// Based on the httprouter package, Copyright 2013 Julien Schmidt.

package chain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/nidorx/chain/pkg"
)

// Router is a high-performance router.
type Router struct {
	registries map[string]*Registry

	contextPool sync.Pool

	Crypto cryptoImpl

	// ConnContext optionally specifies a function that modifies
	// the context used for a new connection c. The provided ctx
	// is derived from the base context and has a ServerContextKey
	// value.
	// ConnContext func(ctx context.Context, c net.Conn) context.Context

	// ReqContext optionally specifies a function that modifies
	// the context used for the request.
	ReqContext func(*Context) context.Context

	// Cached value of global (*) getAllowedHeader methods
	globalAllowed string

	// If enabled, the router automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	// If enabled, the router tries to fix the current request path, if no handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection to the corrected path with status code
	// 301 for GET requests and 308 for all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// Enables automatic redirection if the current route can't be matched but a handler for the path with (without)
	// the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the client is redirected to /foo with http
	// status code 301 for GET requests and 308 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router checks if another method is allowed for the current route, if the current request can not
	// be routed.
	// If this is the case, the request is answered with 'Method Not Allowed' and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFoundHandler handler.
	HandleMethodNotAllowed bool

	// Function to handle panics recovered from http handlers.
	// It should be used to generate a error page and return the http error code 500 (Internal Server Error).
	// The handler can be used to keep your server from crashing because of unrecovered panics.
	PanicHandler func(http.ResponseWriter, *http.Request, any)

	// Function to handle errors recovered from http handlers and middlewares.
	// The handler can be used to do global error handling (not handled in middlewares)
	ErrorHandler func(*Context, error)

	// Configurable http.Handler function which is called when no matching route is found. If it is not set, http.NotFound is
	// used.
	NotFoundHandler http.Handler

	// An optional http.Handler function that is called on automatic OPTIONS requests.
	// The handler is only called if HandleOPTIONS is true and no OPTIONS handler for the specific path was set.
	// The "Allowed" header is set before calling the handler.
	GlobalOPTIONSHandler http.Handler

	// Configurable http.Handler function which is called when a request cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler is called.
	MethodNotAllowedHandler http.Handler
}

func (r *Router) Group(route string) Group {
	return &RouterGroup{p: route, r: r}
}

// GET is a shortcut for router.handleFunc(http.MethodGet, Route, handle)
func (r *Router) GET(route string, handle any) error {
	return r.Handle(http.MethodGet, route, handle)
}

// HEAD is a shortcut for router.handleFunc(http.MethodHead, Route, handle)
func (r *Router) HEAD(route string, handle any) error {
	return r.Handle(http.MethodHead, route, handle)
}

// OPTIONS is a shortcut for router.handleFunc(http.MethodOptions, Route, handle)
func (r *Router) OPTIONS(route string, handle any) error {
	return r.Handle(http.MethodOptions, route, handle)
}

// POST is a shortcut for router.handleFunc(http.MethodPost, Route, handle)
func (r *Router) POST(route string, handle any) error {
	return r.Handle(http.MethodPost, route, handle)
}

// PUT is a shortcut for router.handleFunc(http.MethodPut, Route, handle)
func (r *Router) PUT(route string, handle any) error {
	return r.Handle(http.MethodPut, route, handle)
}

// PATCH is a shortcut for router.handleFunc(http.MethodPatch, Route, handle)
func (r *Router) PATCH(route string, handle any) error {
	return r.Handle(http.MethodPatch, route, handle)
}

// DELETE is a shortcut for router.handleFunc(http.MethodDelete, Route, handle)
func (r *Router) DELETE(route string, handle any) error {
	return r.Handle(http.MethodDelete, route, handle)
}

// Configure allows a RouteConfigurator to perform route configurations
func (r *Router) Configure(route string, configurator RouteConfigurator) {
	configurator.Configure(r, route)
}

var (
	ErrInvalidHandler = errors.New("invalid handler")
	ErrInvalidMethod  = errors.New("method must not be empty")
	ErrInvalidPath    = errors.New("path must begin with '/'")
	ErrHandlerIsNil   = errors.New("handle must not be nil")
)

// Handle registers a new Route for the given method and path.
func (r *Router) Handle(method string, route string, handle any) error {
	method = strings.TrimSpace(method)
	if method == "" {
		return ErrInvalidMethod
	}

	route = pkg.PathClean(route)

	if len(route) < 1 || route[0] != '/' {
		return ErrInvalidPath
	}

	if handle == nil {
		return ErrHandlerIsNil
	}

	if r.registries == nil {
		r.registries = make(map[string]*Registry)
	}

	registry := r.registries[method]
	if registry == nil {
		registry = &Registry{}
		r.registries[method] = registry

		// refresh cache of methods allowed
		r.globalAllowed = r.getAllowedHeader("*", "", nil)
	}

	if handler, err := Handler(handle); err != nil {
		return err
	} else {
		registry.addHandle(route, handler)
	}

	return nil
}

// Handle registers a new Route for the given method and path.
func Handler(handle any) (h Handle, err error) {
	if handler, valid := handle.(Handle); valid {
		h = handler
	} else if handler, valid := handle.(func(*Context) error); valid {
		h = handler
	} else if handler, valid := handle.(func(*Context)); valid {
		h = func(ctx *Context) error {
			handler(ctx)
			return nil
		}
	} else if handler, valid := handle.(http.Handler); valid {
		h = func(ctx *Context) error {
			handler.ServeHTTP(ctx.Writer, ctx.Request)
			return nil
		}
	} else if handler, valid := handle.(http.HandlerFunc); valid {
		h = func(ctx *Context) error {
			handler.ServeHTTP(ctx.Writer, ctx.Request)
			return nil
		}
	} else if handler, valid := handle.(func(w http.ResponseWriter, r *http.Request)); valid {
		h = func(ctx *Context) error {
			handler(ctx.Writer, ctx.Request)
			return nil
		}
	} else if handler, valid := handle.(func(w http.ResponseWriter, r *http.Request) error); valid {
		h = func(ctx *Context) error {
			return handler(ctx.Writer, ctx.Request)
		}
	} else {
		err = ErrInvalidHandler
	}

	return
}

// Use registers a middleware routeT that will match requests with the provided prefix (which is optional and defaults to "/*").
//
//	router.Use(func(ctx *chain.Context) error {
//	    return ctx.NextFunc()
//	})
//
//	router.Use(firstMiddleware, secondMiddleware)
//
//	app.Use("/api", func(ctx *chain.Context) error {
//	    return ctx.NextFunc()
//	})
//
//	app.Use("GET", "/api", func(ctx *chain.Context) error {
//	    return ctx.NextFunc()
//	})
//
//	app.Use("GET", "/files/*filepath", func(ctx *chain.Context) error {
//	    println(ctx.GetParam("filepath"))
//	    return ctx.NextFunc()
//	})
func (r *Router) Use(args ...any) Group {
	var path string
	var methodP string
	var middlewares []func(ctx *Context, next func() error) error

	for i := 0; i < len(args); i++ {
		switch arg := args[i].(type) {
		case string:
			if path == "" {
				path = arg
			} else {
				methodP = path
				path = arg
			}
		case func():
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				arg()
				return next()
			})
		case func() error:
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				if err := arg(); err != nil {
					return err
				}
				return next()
			})
		case func(*Context):
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				arg(ctx)
				return next()
			})
		case func(*Context) error:
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				if err := arg(ctx); err != nil {
					return err
				}
				return next()
			})
		case func(*Context, func() error):
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				arg(ctx, next)
				return nil
			})
		case func(func() error):
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				arg(next)
				return nil
			})
		case func(func() error) error:
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				return arg(next)
			})
		case func(*Context, func() error) error:
			middlewares = append(middlewares, arg)
		case MiddlewareWithInitHandler:
			handler := arg
			handler.Init(methodP, path, r)
			middlewares = append(middlewares, handler.Handle)
		case MiddlewareHandler:
			handler := arg
			middlewares = append(middlewares, handler.Handle)
		case http.Handler:
			// compatibility with http.Handle
			handler := arg
			middlewares = append(middlewares, func(ctx *Context, next func() error) error {
				spy := &ResponseWriterSpy{ResponseWriter: ctx.Writer}
				handler.ServeHTTP(spy, ctx.Request)
				if spy.writeStarted {
					return nil
				}
				return next()
			})
		default:
			panic(fmt.Sprintf("[chain] invalid middleware. middleware: %s", reflect.TypeOf(arg).String()))
		}
	}

	var methods []string

	if methodP == "" || methodP == "*" {
		methods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			//MethodConnect = "CONNECT" ?
			//MethodTrace   = "TRACE" ?
		}
	} else {
		methods = []string{methodP}
	}

	if path == "" || path == "*" {
		path = "/*"
	}

	if r.registries == nil {
		r.registries = make(map[string]*Registry)
	}

	for _, method := range methods {
		registry := r.registries[method]
		if registry == nil {
			registry = &Registry{}
			r.registries[method] = registry
		}
		registry.addMiddleware(path, middlewares)
	}

	return r
}

// Lookup finds the Route and parameters for the given Route and assigns them to the given Context.
func (r *Router) Lookup(method string, path string) (*Route, *Context) {
	if registry := r.registries[method]; registry != nil {
		ctx := r.GetContext(nil, nil, path)
		if route := registry.findHandle(ctx); route != nil {
			return route, ctx
		} else {
			r.PutContext(ctx)
		}
	}
	return nil, nil
}

func (r *Router) updateContext(ctx *Context) *http.Request {
	req := ctx.Request

	// add chain.Context
	reqCtx := ctx.Request.Context()
	reqCtx = context.WithValue(reqCtx, ContextKey, ctx)
	req = req.WithContext(reqCtx)
	ctx.Request = req

	// add custom context
	if rc := r.ReqContext; rc != nil {
		if reqCtx := rc(ctx); reqCtx != nil {
			req = req.WithContext(reqCtx)
			ctx.Request = req
		}
	}
	return req
}

// ServeHTTP responds to the given request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	rw := &ResponseWriterSpy{ResponseWriter: w}
	w = rw
	var ctx *Context

	defer func() {
		if rcv := recover(); rcv != any(nil) {
			if r.PanicHandler != nil {
				r.PanicHandler(w, req, rcv)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else if !rw.writeStarted && ctx != nil {
			// if necessary, write header on exit
			ctx.write()
		}

		// execute after write hooks
		rw.execAfterWriteHooksCalledByRouter()
	}()

	ctx = r.GetContext(req, w, "")

	go func() {
		// clear context when connection is closed
		<-ctx.Request.Context().Done()
		r.PutContext(ctx)
	}()

	path := req.URL.Path

	if registry := r.registries[req.Method]; registry != nil {
		if route := registry.findHandle(ctx); route != nil {
			ctx.MatchedRoutePath = route.Path.path
			r.updateContext(ctx)
			if err := route.Dispatch(ctx); err != nil {
				if r.ErrorHandler != nil {
					r.ErrorHandler(ctx, err)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
			return
		} else if req.Method != http.MethodConnect && path != "/" {
			// Moved Permanently, request with GET method
			code := http.StatusMovedPermanently
			if req.Method != http.MethodGet {
				// Permanent Redirect, request with same method
				code = http.StatusPermanentRedirect
			}

			if r.RedirectTrailingSlash {
				// checks if it has a route that allows redirection
				tsrPath := path
				if len(tsrPath) > 1 && tsrPath[len(tsrPath)-1] == '/' {
					tsrPath = tsrPath[:len(tsrPath)-1]
				} else {
					tsrPath = tsrPath + "/"
				}
				ctx2 := &Context{path: tsrPath}
				ctx2.parsePathSegments()
				if tsr := registry.findHandle(ctx2); tsr != nil {
					req.URL.Path = tsrPath
					http.Redirect(w, req, req.URL.String(), code)
					return
				}
			}

			// Try to fix the request path
			if r.RedirectFixedPath {
				ctx2 := &Context{path: pkg.PathClean(path)}
				ctx2.parsePathSegments()
				if fixed := registry.findHandleCaseInsensitive(ctx2); fixed != nil {
					req.URL.Path = fixed.Path.ReplacePath(ctx2)
					http.Redirect(w, req, req.URL.String(), code)
					return
				} else if r.RedirectTrailingSlash {
					tsrPath := ctx2.path
					if len(tsrPath) > 1 && tsrPath[len(tsrPath)-1] == '/' {
						tsrPath = tsrPath[:len(tsrPath)-1]
					} else {
						tsrPath = tsrPath + "/"
					}
					ctx2 = &Context{path: tsrPath}
					ctx2.parsePathSegments()
					if fixed = registry.findHandleCaseInsensitive(ctx2); fixed != nil {
						req.URL.Path = fixed.Path.ReplacePath(ctx2)
						http.Redirect(w, req, req.URL.String(), code)
						return
					}
				}
			}
		}
	}

	req = r.updateContext(ctx)
	if req.Method == http.MethodOptions && r.HandleOPTIONS {
		// Handle OPTIONS requests
		if allow := r.getAllowedHeader(path, http.MethodOptions, ctx); allow != "" {
			w.Header().Set("Allow", allow)
			if r.GlobalOPTIONSHandler != nil {
				r.GlobalOPTIONSHandler.ServeHTTP(w, req)
			}
			return
		}
	} else if r.HandleMethodNotAllowed { // Handle 405
		if allow := r.getAllowedHeader(path, req.Method, ctx); allow != "" {
			w.Header().Set("Allow", allow)
			if r.MethodNotAllowedHandler != nil {
				r.MethodNotAllowedHandler.ServeHTTP(w, req)
			} else {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			}
			return
		}
	}

	// Handle 404
	if r.NotFoundHandler != nil {
		r.NotFoundHandler.ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
}

// GetContext returns a new ContextImpl from the pool.
func (r *Router) GetContext(req *http.Request, w http.ResponseWriter, path string) *Context {
	ctx := r.contextPool.Get().(*Context)
	ctx.router = r
	ctx.Writer = w
	ctx.Request = req
	ctx.paramCount = 0

	if req != nil {
		ctx.path = req.URL.Path
	} else {
		ctx.path = path
	}
	ctx.parsePathSegments()
	return ctx
}

// PutContext Close frees up resources and is automatically called in the ServeHTTP part of the web server.
func (r *Router) PutContext(ctx *Context) {
	if ctx.children != nil {
		for _, child := range ctx.children {
			r.PutContext(child)
		}
		ctx.children = nil
	}
	ctx.router = nil
	ctx.Writer = nil
	ctx.Request = nil
	ctx.data = nil
	ctx.root = nil
	r.contextPool.Put(ctx)
}

func (r *Router) getAllowedHeader(path string, reqMethod string, ctx *Context) (allow string) {
	allowed := make([]string, 0, 9)

	if path == "*" {
		// server-wide
		// empty method is used for internal calls to refresh the cache
		if reqMethod == "" {
			for method := range r.registries {
				if method == http.MethodOptions {
					continue
				}
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		} else {
			return r.globalAllowed
		}
	} else { // specific path
		for method, registry := range r.registries {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == http.MethodOptions {
				continue
			}

			if route := registry.findHandle(ctx); route != nil {
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		}
	}

	if len(allowed) > 0 {
		// Add request method to list of allowed methods
		allowed = append(allowed, http.MethodOptions)

		// Sort allowed methods.
		// sort.Strings(allowed) unfortunately causes unnecessary allocations
		// due to allowed being moved to the heap and interface conversion
		for i, l := 1, len(allowed); i < l; i++ {
			for j := i; j > 0 && allowed[j] < allowed[j-1]; j-- {
				allowed[j], allowed[j-1] = allowed[j-1], allowed[j]
			}
		}

		// return as comma separated list
		return strings.Join(allowed, ", ")
	}

	return allow
}
