package main

import (
	"embed"
	"fmt"
	"github.com/syntax-framework/chain"
	"github.com/syntax-framework/chain/socket"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	//go:embed public
	staticFiles embed.FS
	staticDir   = "public"
)

func main() {
	router := chain.New()

	var staticFS = http.FS(staticFiles)
	fs := rootPath(http.FileServer(staticFS))
	router.GET("/*", fs)

	router.Configure("/socket", AppSocket)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s...\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), router); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

func rootPath(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = fmt.Sprintf("/%s/", staticDir)
		} else {
			b := strings.Split(r.URL.Path, "/")[0]
			if b != staticDir {
				r.URL.Path = fmt.Sprintf("/%s%s", staticDir, r.URL.Path)
			}
		}
		h.ServeHTTP(w, r)
	})
}

var AppSocket = &socket.Handler{
	Channels: []*socket.Channel{
		socket.NewChannel("chat:*", chatChannel),
	},
	OnConfig: func(handler *socket.Handler, router *chain.Router, endpoint string) error {
		return nil
	},
	OnConnect: func(info *socket.Session) error {
		return nil
	},
}

func chatChannel(channel *socket.Channel) {

	channel.Join("chat:lobby", func(_ any, _ *socket.Socket) (reply any, err error) {
		return
	})

	channel.HandleIn("shout", func(event string, payload any, socket *socket.Socket) (reply any, err error) {
		err = socket.Broadcast("shout", payload)
		return
	})
}
