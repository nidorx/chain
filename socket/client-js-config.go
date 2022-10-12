package socket

import (
	"embed"
	"github.com/rs/zerolog/log"
	"github.com/syntax-framework/chain"
)

var (
	//go:embed client/chain.js
	clientJsFS             embed.FS
	clientJsContent        []byte
	configuredRouterClient = map[*chain.Router]bool{}
)

func init() {
	var err error
	if clientJsContent, err = clientJsFS.ReadFile("client/chain.js"); err != nil {
		log.Panic().Err(err).Caller(1).
			Stack().
			Msg(_l("cannot load client/chain.js"))
	}
}

func clientJsAddHandler(router *chain.Router) {
	if _, exist := configuredRouterClient[router]; exist {
		return
	}

	router.GET("/syntax-chain.js", func(ctx *chain.Context) {
		ctx.SetHeader("Content-Type", "application/javascript")
		// Content-Length
		// Etag
		// Last-Modified
		if _, err := ctx.Write(clientJsContent); err != nil {
			log.Error().Stack().Err(err).Caller(1).
				Msg(_l("it was not possible to deliver /syntax-chain.js"))
		}
	})
}
