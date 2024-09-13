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
	var next func() error
	next = func() error {
		if index > len(r.Middlewares)-1 {
			// end of middlewares
			return r.Handle(ctx)
		}

		middleware := r.Middlewares[index]
		index++

		match, names, values := middleware.Path.Match(ctx)
		if match {
			var nextErr error
			calledNext := false
			nextMid := func() error {
				if calledNext {
					slog.Warn(
						"[chain] calling next() multiple times for route",
						slog.Int("index", index),
						slog.String("path", ctx.path),
					)

					return nextErr
				}
				calledNext = true
				nextErr = next()
				return nextErr
			}

			if len(names) > 0 {
				// middleware expects parameterizable route
				return middleware.Handle(ctx.WithParams(names, values), nextMid)
			} else {
				// use same context
				return middleware.Handle(ctx, nextMid)
			}
		}
		return next()
	}
	return next()
}
