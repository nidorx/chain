package chain

import (
	"context"
	"net/http"
)

type chainContextKey struct{}
type bodyBytesKey struct{}

// ContextKey is the request context key under which URL params are stored.
var ContextKey = chainContextKey{}

// BodyBytesKey indicates a default body bytes key.
var BodyBytesKey = bodyBytesKey{}

// GetContext pulls the URL parameters from a request context, or returns nil if none are present.
func GetContext(ctx context.Context) *Context {
	p, _ := ctx.Value(ContextKey).(*Context)
	return p
}

// Context represents a request & response Context.
type Context struct {
	paramCount        int
	pathSegmentsCount int
	pathSegments      [32]int
	path              string
	paramNames        [32]string
	paramValues       [32]string
	data              map[any]any
	handler           Handle
	router            *Router
	Route             *RouteInfo
	Writer            http.ResponseWriter
	Request           *http.Request
	Crypto            *cryptoImpl
	parent            *Context
	index             int
	children          []*Context
}

// Set define um valor compartilhado no contexto de execução da requisição
func (ctx *Context) Set(key any, value any) {
	if ctx.data == nil {
		ctx.data = make(map[any]any)
	}
	ctx.data[key] = value
}

// Get obtém um valor compartilhado no contexto de execução da requisição
func (ctx *Context) Get(key any) (any, bool) {
	if ctx.data != nil {
		value, exists := ctx.data[key]
		if exists {
			return value, exists
		}
	}

	if ctx.parent != nil {
		return ctx.parent.Get(key)
	}
	return nil, false
}

func (ctx *Context) Destroy() {
	if ctx.parent == nil {
		// root context, will be removed automaticaly
		return
	}
	if ctx.parent.children != nil {
		ctx.parent.children[ctx.index] = nil
	}
	ctx.parent = nil
	ctx.children = nil

	if ctx.router != nil {
		ctx.router.poolPutContext(ctx)
	}
}

func (ctx *Context) Child() *Context {
	var child *Context
	if ctx.router != nil {
		child = ctx.router.poolGetContext(ctx.Request, ctx.Writer, "")
	} else {
		child = &Context{
			path:    ctx.path,
			Crypto:  crypt,
			Writer:  ctx.Writer,
			Request: ctx.Request,
			handler: ctx.handler,
		}
	}

	child.paramCount = ctx.paramCount
	child.paramNames = ctx.paramNames
	child.paramValues = ctx.paramValues
	child.pathSegments = ctx.pathSegments
	child.pathSegmentsCount = ctx.pathSegmentsCount
	child.Route = ctx.Route

	child.parent = ctx

	if ctx.children == nil {
		ctx.children = make([]*Context, 0)
	}
	child.index = len(ctx.children)
	ctx.children = append(ctx.children, child)

	return child
}

// func (ctx *Context) With(key any, value any) *Context {

// }

func (ctx *Context) WithParams(names []string, values []string) *Context {
	child := ctx.Child()
	child.paramCount = len(names)
	child.paramNames = [32]string{}
	child.paramValues = [32]string{}

	for i, name := range ctx.paramNames {
		child.paramNames[i] = name
		child.paramValues[i] = ctx.paramValues[i]
	}

	for i := 0; i < len(names); i++ {
		child.paramNames[i] = names[i]
		child.paramValues[i] = values[i]
	}

	return child
}

// NewUID get a new KSUID.
//
// KSUID is for K-Sortable Unique IDentifier. It is a kind of globally unique identifier similar to a RFC 4122 UUID,
// built from the ground-up to be "naturally" sorted by generation timestamp without any special type-aware logic.
//
// See: https://github.com/segmentio/ksuid
func (ctx *Context) NewUID() (uid string) {
	return NewUID()
}

// Router get current router reference
func (ctx *Context) Router() *Router {
	return ctx.router
}

// BeforeSend Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (ctx *Context) BeforeSend(callback func()) error {
	if spy, is := ctx.Writer.(*ResponseWriterSpy); is {
		return spy.beforeWriteHeader(callback)
	}
	return nil
}

func (ctx *Context) AfterSend(callback func()) error {
	if spy, is := ctx.Writer.(*ResponseWriterSpy); is {
		return spy.afterWrite(callback)
	}
	return nil
}

func (ctx *Context) write() {
	if spy, is := ctx.Writer.(*ResponseWriterSpy); is {
		if !spy.writeStarted {
			ctx.WriteHeader(http.StatusOK)
		}
	}
}

// addParameter adds a new parameter to the Context.
func (ctx *Context) addParameter(name string, value string) {
	ctx.paramNames[ctx.paramCount] = name
	ctx.paramValues[ctx.paramCount] = value
	ctx.paramCount++
}

func (ctx *Context) parsePathSegments() {
	ctx.pathSegmentsCount = parsePathSegments(ctx.path, &ctx.pathSegments)
}
