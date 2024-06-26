package chain

import (
	"context"
	"net/http"
	"strings"
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
	MatchedRoutePath  string
	Writer            http.ResponseWriter
	Request           *http.Request
	Crypto            *cryptoImpl
	root              *Context
	children          []*Context
}

// Set define um valor compartilhado no contexto de execução da requisição
func (ctx *Context) Set(key any, value any) {
	if ctx.root != nil {
		ctx.root.Set(key, value)
	}
	if ctx.data == nil {
		ctx.data = make(map[any]any)
	}
	ctx.data[key] = value
}

// Get obtém um valor compartilhado no contexto de execução da requisição
func (ctx *Context) Get(key any) (any, bool) {
	if ctx.root != nil {
		return ctx.root.Get(key)
	}

	if ctx.data == nil {
		return nil, false
	}
	value, exists := ctx.data[key]
	return value, exists
}

func (ctx *Context) WithParams(names []string, values []string) *Context {
	var child *Context
	if ctx.router != nil {
		child = ctx.router.GetContext(ctx.Request, ctx.Writer, "")
	} else {
		child = &Context{
			Writer:      ctx.Writer,
			Request:     ctx.Request,
			handler:     ctx.handler,
			paramCount:  len(names),
			paramNames:  ctx.paramNames,
			paramValues: ctx.paramValues,
		}
	}
	for i := 0; i < len(names); i++ {
		child.paramNames[i] = names[i]
		child.paramValues[i] = values[i]
	}

	if ctx.root == nil {
		child.root = ctx
	} else {
		child.root = ctx.root
	}

	if child.root.children == nil {
		child.root.children = make([]*Context, 0)
	}
	child.root.children = append(child.root.children, child)

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
	var (
		segmentStart = 0
		segmentSize  int
		path         = ctx.path
	)
	if len(path) > 0 {
		path = path[1:]
	}

	ctx.pathSegments[0] = 0
	ctx.pathSegmentsCount = 1

	for {
		segmentSize = strings.IndexByte(path, separator)
		if segmentSize == -1 {
			segmentSize = len(path)
		}
		ctx.pathSegments[ctx.pathSegmentsCount] = segmentStart + 1 + segmentSize

		if segmentSize == len(path) {
			break
		}
		ctx.pathSegmentsCount++
		path = path[segmentSize+1:]
		segmentStart = segmentStart + 1 + segmentSize
	}
}
