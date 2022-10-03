package chain

import "embed"

//go:embed socket/client/chain.js
var socketClientJS embed.FS

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

	router.GET("/syntax-chain.js", func(ctx *Context) {
		clientJsBytes, _ := socketClientJS.ReadFile("socket/client/chain.js")
		ctx.Header().Set("Content-Type", "application/javascript")
		//"Content-Range": {r.contentRange(size)},
		//"Content-Type":  {contentType},
		// Content-Length
		// Etag
		// Last-Modified
		ctx.Write(clientJsBytes)
	})

	return router
}
