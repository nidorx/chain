package chain

type Group interface {
	GET(route string, handle any) error
	HEAD(route string, handle any) error
	OPTIONS(route string, handle any) error
	POST(route string, handle any) error
	PUT(route string, handle any) error
	PATCH(route string, handle any) error
	DELETE(route string, handle any) error
	Use(args ...any) (Group, error)
	Group(route string) Group
	Handle(method string, route string, handle any) error
	Configure(route string, configurator RouteConfigurator)
}

type RouterGroup struct {
	path   string
	router *Router
}

func (r *RouterGroup) GET(route string, handle any) error {
	return r.router.GET(r.path+route, handle)
}

func (r *RouterGroup) HEAD(route string, handle any) error {
	return r.router.HEAD(r.path+route, handle)
}

func (r *RouterGroup) OPTIONS(route string, handle any) error {
	return r.router.OPTIONS(r.path+route, handle)
}

func (r *RouterGroup) POST(route string, handle any) error {
	return r.router.POST(r.path+route, handle)
}

func (r *RouterGroup) PUT(route string, handle any) error {
	return r.router.PUT(r.path+route, handle)
}

func (r *RouterGroup) PATCH(route string, handle any) error {
	return r.router.PATCH(r.path+route, handle)
}

func (r *RouterGroup) DELETE(route string, handle any) error {
	return r.router.DELETE(r.path+route, handle)
}

// Use registers a middleware routeT that will match requests with the provided prefix (which is optional and defaults to "/*").
//
//	group.Use(func(ctx *chain.Context) error {
//	    return ctx.NextFunc()
//	})
//
//	group.Use(firstMiddleware, secondMiddleware)
//
//	group.Use("/api", func(ctx *chain.Context) error {
//	    return ctx.NextFunc()
//	})
//
//	group.Use("GET", "/api", func(ctx *chain.Context) error {
//	    return ctx.NextFunc()
//	})
//
//	group.Use("GET", "/files/*filepath", func(ctx *chain.Context) error {
//	    println(ctx.GetParam("filepath"))
//	    return ctx.NextFunc()
//	})
func (r *RouterGroup) Use(args ...any) (Group, error) {

	var (
		path    string
		method  string
		newArgs = []any{"", ""}
	)

	for i := range args {
		switch arg := args[i].(type) {
		case string:
			if path == "" {
				// group.Use("/api", func...)
				path = arg
			} else {
				// group.Use("GET", "/api", func...)
				method = path
				path = arg
			}
		default:
			newArgs = append(newArgs, arg)
		}
	}

	if method == "" {
		method = "*"
	}

	if path == "" || path == "*" {
		path = "/*"
	}

	path = r.path + path

	newArgs[0] = method
	newArgs[1] = path

	if _, err := r.router.Use(newArgs...); err != nil {
		return r, err
	}

	return r, nil
}

func (r *RouterGroup) Group(route string) Group {
	return &RouterGroup{r.path + route, r.router}
}

func (r *RouterGroup) Handle(method string, route string, handle any) error {
	return r.router.Handle(method, r.path+route, handle)
}

func (r *RouterGroup) Configure(route string, configurator RouteConfigurator) {
	r.router.Configure(r.path+route, configurator)
}
