package chain

import (
	"fmt"
	"log/slog"
	"strings"
)

// Registry is an algorithm-independent framework for recording routes. This division allows us to explore different
// algorithms without breaking the contract.
type Registry struct {
	canBeStatic [2048]bool
	storage     *RouteStorage
	routes      []*Route
	middlewares []*Middleware
	static      map[string]*Route
}

func (r *Registry) findHandle(ctx *Context) *Route {
	if r.canBeStatic[len(ctx.path)] {
		if route, found := r.static[ctx.path]; found {
			return route
		}
	}

	if r.storage == nil {
		return nil
	}

	return r.storage.lookup(ctx)
}

func (r *Registry) findHandleCaseInsensitive(ctx *Context) *Route {
	if r.canBeStatic[len(ctx.path)] {
		for key, route := range r.static {
			if strings.EqualFold(ctx.path, key) {
				return route
			}
		}
	}

	if r.storage == nil {
		return nil
	}

	return r.storage.lookupCaseInsensitive(ctx)
}

func (r *Registry) addHandle(path string, handle Handle) error {
	if r.routes == nil {
		r.routes = []*Route{}
	}

	details, err := ParseRouteInfo(path)
	if err != nil {
		return err
	}

	// avoid conflicts
	for _, route := range r.routes {
		if details.conflictsWith(route.Info) {
			slog.Error("[chain] wildcard conflicts", slog.String("new", details.path), slog.String("existing", route.Info.path))
			return fmt.Errorf("[chain] wildcard conflicts: new=%s existing=%s", details.path, route.Info.path)
		}
	}

	if !details.hasParameter && !details.hasWildcard {
		if r.static == nil {
			r.static = map[string]*Route{}
		}

		r.canBeStatic[len(path)] = true
		r.static[path] = r.createRoute(handle, details)
		return nil
	}

	if r.storage == nil {
		r.storage = &RouteStorage{}
	}

	r.storage.add(r.createRoute(handle, details))
	return nil
}

func (r *Registry) createRoute(handle Handle, info *RouteInfo) *Route {
	route := &Route{
		Handle:           handle,
		Info:             info,
		middlewaresAdded: map[*Middleware]bool{},
	}

	r.routes = append(r.routes, route)

	for _, middleware := range r.middlewares {
		if route.middlewaresAdded[middleware] != true && middleware.Path.Matches(route.Info) {
			route.middlewaresAdded[middleware] = true
			route.Middlewares = append(route.Middlewares, middleware)
		}
	}

	return route
}

func (r *Registry) addMiddleware(path string, middlewares []func(ctx *Context, next func() error) error) error {
	if r.middlewares == nil {
		r.middlewares = []*Middleware{}
	}

	for _, middleware := range middlewares {
		info, err := ParseRouteInfo(path)
		if err != nil {
			return err
		}

		mw := &Middleware{
			Path:   info,
			Handle: middleware,
		}

		r.middlewares = append(r.middlewares, mw)

		// add this MiddlewareFunc to all compatible routes
		for _, route := range r.routes {
			if route.middlewaresAdded[mw] != true && mw.Path.Matches(route.Info) {
				route.middlewaresAdded[mw] = true
				route.Middlewares = append(route.Middlewares, mw)
			}
		}
	}
	return nil
}
