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
	clientModTime, _       = time.Parse(time.DateTime, "2025-02-09 00:00:00")
	configuredRouterClient = map[*chain.Router]bool{}
)

var (
	//go:embed client/chain.ts
	clientTsFS      embed.FS
	clientTsEtag    string
	clientTsContent []byte
)

var (
	//go:embed client/chain.js
	clientJsFS      embed.FS
	clientJsEtag    string
	clientJsContent []byte
)

var (
	//go:embed client/chain.js.map
	clientJsMapFS      embed.FS
	clientJsMapEtag    string
	clientJsMapContent []byte
)

func init() {
	if content, err := clientTsFS.ReadFile("client/chain.ts"); err != nil {
		panic(fmt.Sprintf("[chain] cannot load client/chain.ts. Error: %s", err.Error()))
	} else {
		clientTsContent = content
		clientTsEtag = chain.HashCrc32(clientJsContent)
	}

	if content, err := clientJsFS.ReadFile("client/chain.js"); err != nil {
		panic(fmt.Sprintf("[chain] cannot load client/chain.js. Error: %s", err.Error()))
	} else {
		clientJsContent = content
		clientJsEtag = chain.HashCrc32(clientJsContent)
	}

	if content, err := clientJsMapFS.ReadFile("client/chain.js.map"); err != nil {
		panic(fmt.Sprintf("[chain] cannot load client/chain.js. Error: %s", err.Error()))
	} else {
		clientJsMapContent = content
		clientJsMapEtag = chain.HashCrc32(clientJsContent)
	}
}

// ClientJsHandler add "/chain.js" endpoint
func ClientJsHandler(r *chain.Router, route string) {
	if _, exist := configuredRouterClient[r]; exist {
		return
	}

	r.GET("/chain.ts", func(ctx *chain.Context) {
		ctx.SetHeader("ETag", clientTsEtag)
		ctx.SetHeader("Content-Type", "application/typescript; charset=utf-8")
		ctx.SetHeader("Content-Length", strconv.Itoa(len(clientTsContent)))
		http.ServeContent(ctx.Writer, ctx.Request, "/chain.ts", clientModTime, bytes.NewReader(clientTsContent))
	})

	r.GET("/chain.js", func(ctx *chain.Context) {
		ctx.SetHeader("ETag", clientJsEtag)
		ctx.SetHeader("Content-Type", "text/javascript; charset=utf-8")
		ctx.SetHeader("Content-Length", strconv.Itoa(len(clientJsContent)))
		http.ServeContent(ctx.Writer, ctx.Request, "/chain.js", clientModTime, bytes.NewReader(clientJsContent))
	})

	r.GET("/chain.js.map", func(ctx *chain.Context) {
		ctx.SetHeader("ETag", clientJsMapEtag)
		ctx.SetHeader("Content-Type", "application/json")
		ctx.SetHeader("Content-Length", strconv.Itoa(len(clientJsMapContent)))
		http.ServeContent(ctx.Writer, ctx.Request, "/chain.js.map", clientModTime, bytes.NewReader(clientJsMapContent))
	})
}
