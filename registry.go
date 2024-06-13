package chain

import (
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

func (r *Registry) addHandle(path string, handle Handle) {
	if r.routes == nil {
		r.routes = []*Route{}
	}

	details := ParsePathDetails(path)

	// avoid conflicts
	for _, route := range r.routes {
		if details.conflictsWith(route.Path) {
			slog.Error("[chain] wildcard conflicts", slog.String("new", details.path), slog.String("existing", route.Path.path))
			panic("[chain] wildcard conflicts")
		}
	}

	if !details.hasParameter && !details.hasWildcard {
		if r.static == nil {
			r.static = map[string]*Route{}
		}

		r.canBeStatic[len(path)] = true
		r.static[path] = r.createRoute(handle, details)
		return
	}

	if r.storage == nil {
		r.storage = &RouteStorage{}
	}

	r.storage.add(r.createRoute(handle, details))
}

func (r *Registry) createRoute(handle Handle, pathDetails *PathDetails) *Route {
	route := &Route{
		Handle:           handle,
		Path:             pathDetails,
		middlewaresAdded: map[*Middleware]bool{},
	}

	r.routes = append(r.routes, route)

	for _, middleware := range r.middlewares {
		if route.middlewaresAdded[middleware] != true && middleware.Path.MaybeMatches(route.Path) {
			route.middlewaresAdded[middleware] = true
			route.Middlewares = append(route.Middlewares, middleware)
		}
	}

	return route
}

func (r *Registry) addMiddleware(path string, middlewares []func(ctx *Context, next func() error) error) {
	if r.middlewares == nil {
		r.middlewares = []*Middleware{}
	}

	for _, middleware := range middlewares {
		info := &Middleware{
			Path:   ParsePathDetails(path),
			Handle: middleware,
		}

		r.middlewares = append(r.middlewares, info)

		// add this MiddlewareFunc to all compatible routes
		for _, route := range r.routes {
			if route.middlewaresAdded[info] != true && info.Path.MaybeMatches(route.Path) {
				route.middlewaresAdded[info] = true
				route.Middlewares = append(route.Middlewares, info)
			}
		}
	}
}
