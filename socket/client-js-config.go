package socket

import (
	"bytes"
	"embed"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/nidorx/chain"
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
		panic(fmt.Sprintf("[chain] cannot load client/chain.js. Error: %s", err.Error()))
	} else {
		clientJsContent = content
		clientJsEtag = chain.HashCrc32(clientJsContent)
	}
}

// ClientJsHandler add "/chain.js" endpoint
func ClientJsHandler(r *chain.Router, route string) {
	if _, exist := configuredRouterClient[r]; exist {
		return
	}

	r.GET("/chain.js", func(ctx *chain.Context) {
		ctx.SetHeader("Content-Type", "text/javascript; charset=utf-8")
		ctx.SetHeader("Content-Length", strconv.Itoa(len(clientJsContent)))
		ctx.SetHeader("ETag", clientJsEtag)
		http.ServeContent(ctx.Writer, ctx.Request, "/chain.js", clientJsModTime, bytes.NewReader(clientJsContent))
	})
}
