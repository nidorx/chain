package chain

import (
	"log/slog"
)

const (
	separator = '/'
	parameter = ':'
	wildcard  = '*'
)

type RouteConfigurator interface {
	Configure(router *Router, path string)
}

type MiddlewareHandler interface {
	Handle(ctx *Context, next func() error) error
}

type MiddlewareWithInitHandler interface {
	Init(method string, path string, router *Router)
	Handle(ctx *Context, next func() error) error
}

type Handle func(*Context) error

type Middleware struct {
	Path   *RouteInfo
	Handle func(ctx *Context, next func() error) error
}

// Route control of a registered route
type Route struct {
	Info             *RouteInfo
	Handle           Handle
	Middlewares      []*Middleware
	middlewaresAdded map[*Middleware]bool
}

// Dispatch ctx into this route
func (r *Route) Dispatch(ctx *Context) error {
	if len(r.Middlewares) == 0 {
		return r.Handle(ctx)
	}

	index := 0
	currentCtx := ctx // track the current context through the middleware chain
	var next func() error
	next = func() error {
		if index > len(r.Middlewares)-1 {
			// end of middlewares
			return r.Handle(currentCtx)
		}

		middleware := r.Middlewares[index]
		index++

		match, names, values := middleware.Path.Match(currentCtx)
		if match {
			var nextErr error
			calledNext := false
			nextMid := func() error {
				if calledNext {
					slog.Warn(
						"[chain] calling next() multiple times for route",
						slog.Int("index", index),
						slog.String("path", currentCtx.path),
					)

					return nextErr
				}
				calledNext = true
				nextErr = next()
				return nextErr
			}

			if len(names) > 0 {
				// middleware expects parameterizable route
				currentCtx = currentCtx.WithParams(names, values)
				return middleware.Handle(currentCtx, nextMid)
			} else {
				// use same context
				return middleware.Handle(currentCtx, nextMid)
			}
		}
		return next()
	}
	return next()
}
