package socket

import (
	"bytes"
	"embed"
	"github.com/nidorx/chain"
	"github.com/rs/zerolog/log"
	"net/http"
	"path"
	"strconv"
	"time"
)

var (
	//go:embed client/chain.js
	clientJsFS             embed.FS
	clientJsContent        []byte
	clientJsEtag           string
	clientJsModTime, _     = time.Parse(time.DateTime, "2023-05-07 00:00:00")
	configuredRouterClient = map[*chain.Router]bool{}
)

func init() {
	if content, err := clientJsFS.ReadFile("client/chain.js"); err != nil {
		log.Panic().Err(err).Caller(1).Stack().Msg(_l("cannot load client/chain.js"))
	} else {
		clientJsContent = content
		clientJsEtag = chain.HashCrc32(clientJsContent)
	}
}

// ClientJsHandler add "/chain.js" endpoint
func ClientJsHandler(r *chain.Router, route string) {
	if _, exist := configuredRouterClient[r]; exist {
		// @TODO: Permitir saber quando a instancia é destruida r.OnDestroy(func() { })
		return
	}

	jsPath := path.Join(route, "/chain.js")
	r.GET(jsPath, func(ctx *chain.Context) {
		ctx.SetHeader("Content-Type", "text/javascript; charset=utf-8")
		ctx.SetHeader("Content-Length", strconv.Itoa(len(clientJsContent)))
		ctx.SetHeader("ETag", clientJsEtag)
		http.ServeContent(ctx.Writer, ctx.Request, "/chain.js", clientJsModTime, bytes.NewReader(clientJsContent))
	})
}
