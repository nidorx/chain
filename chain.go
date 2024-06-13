package chain

func New() *Router {
	router := &Router{
		HandleOPTIONS:          true,
		RedirectFixedPath:      true,
		RedirectTrailingSlash:  true,
		HandleMethodNotAllowed: true,
	}
	router.contextPool.New = func() any {
		return &Context{}
	}

	return router
}
