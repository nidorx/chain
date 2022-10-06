package chain

import "github.com/rs/zerolog/log"

var logger = log.With().Str("package", "chain.router").Logger()

func New() *Router {
	router := &Router{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}
	router.contextPool.New = func() any {
		return &Context{}
	}

	return router
}
