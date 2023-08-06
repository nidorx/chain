package chain

type Group interface {
	GET(route string, handle any)
	HEAD(route string, handle any)
	OPTIONS(route string, handle any)
	POST(route string, handle any)
	PUT(route string, handle any)
	PATCH(route string, handle any)
	DELETE(route string, handle any)
	Use(args ...any) Group
	Group(route string) Group
	Handle(method string, route string, handle any)
	Configure(route string, configurator RouteConfigurator)
}

type RouterGroup struct {
	p string
	r *Router
}

func (r *RouterGroup) GET(route string, handle any)     { r.r.GET(r.p+route, handle) }
func (r *RouterGroup) HEAD(route string, handle any)    { r.r.HEAD(r.p+route, handle) }
func (r *RouterGroup) OPTIONS(route string, handle any) { r.r.OPTIONS(r.p+route, handle) }
func (r *RouterGroup) POST(route string, handle any)    { r.r.POST(r.p+route, handle) }
func (r *RouterGroup) PUT(route string, handle any)     { r.r.PUT(r.p+route, handle) }
func (r *RouterGroup) PATCH(route string, handle any)   { r.r.PATCH(r.p+route, handle) }
func (r *RouterGroup) DELETE(route string, handle any)  { r.r.DELETE(r.p+route, handle) }
func (r *RouterGroup) Use(args ...any) Group            { return r.r.Use(args...) }
func (r *RouterGroup) Group(route string) Group         { return &RouterGroup{r.p + route, r.r} }
func (r *RouterGroup) Handle(method string, route string, handle any) {
	r.r.Handle(method, r.p+route, handle)
}
func (r *RouterGroup) Configure(route string, configurator RouteConfigurator) {
	r.r.Configure(r.p+route, configurator)
}
