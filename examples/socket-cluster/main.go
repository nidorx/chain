package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"math/rand"

	"github.com/nidorx/chain"
	"github.com/nidorx/chain/pubsub"
	"github.com/nidorx/chain/socket"
)

var (
	//go:embed public
	staticFs     embed.FS
	staticDir    = "public"
	lastPortUsed = 8080
	cluster      = map[int]*http.Server{}
)

func main() {

	// Used by session
	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}

	initPublisher()

	router := chain.New()
	router.GET("/*", createStaticFileHandler())
	router.GET("/node", listNodeHandler)
	router.POST("/node", addNodeHandler)
	router.DELETE("/node", deleteNodeHandler)
	socket.ClientJsHandler(router, "/") // "/chain.js"

	port := fmt.Sprintf("%d", lastPortUsed)
	log.Printf("Listening on :%s...\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), router); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

func initPublisher() {
	// send message to all nodes, using pubsub (local)
	go func() {
		addNodeHandler(nil) // first socket server
		addNodeHandler(nil) // second socket server

		ticker := time.NewTicker(500 * time.Millisecond)
		i := 1
		for range ticker.C {
			i++
			if bytes, err := json.Marshal(map[string]any{"name": "Server", "body": fmt.Sprintf("msg %d", i)}); err != nil {
				return
			} else {
				pubsub.Broadcast("chat:lobby", bytes)
			}
		}
	}()
}

func deleteNodeHandler(ctx *chain.Context) {
	if len(cluster) > 0 {
		for {
			toRemove := 0
			for port := range cluster {
				if rand.Intn(100) > 50 {
					toRemove = port
					break
				}
			}
			if toRemove > 0 {
				sctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
				if err := cluster[toRemove].Shutdown(sctx); err != nil {
					cluster[toRemove].Close()
				}
				delete(cluster, toRemove)
				break
			}
		}
	}
	ctx.OK()
}

func listNodeHandler(ctx *chain.Context) {
	var nodes []int
	for port, _ := range cluster {
		nodes = append(nodes, port)
	}
	ctx.Json(nodes)
}

func addNodeHandler(ctx *chain.Context) {
	lastPortUsed++
	port := fmt.Sprintf("%d", lastPortUsed)

	channel := socket.NewChannel("chat:*", func(channel *socket.Channel) {

		channel.Join("chat:lobby", func(payload any, skt *socket.Socket) (reply any, err error) {
			slog.Info(
				skt.Topic()+" join",
				slog.String("server", port),
				slog.Any("socket", skt.Id()),
				slog.Any("payload", payload),
			)
			return
		})

		channel.Leave("chat:lobby", func(skt *socket.Socket, reason socket.LeaveReason) {
			slog.Info(
				skt.Topic()+" leave",
				slog.String("server", port),
				slog.Any("socket", skt.Id()),
				slog.Any("reason", reason),
			)
		})

		// automatically send messages to connected clients (from pubsub)
		channel.Subscribe("chat:*", "message")
	})

	handler := &socket.Handler{
		Channels: []*socket.Channel{channel},
		OnConfig: func(handler *socket.Handler, router *chain.Router, endpoint string) error {
			return nil
		},
		OnConnect: func(info *socket.Session) error {
			return nil
		},
		Transports: []socket.Transport{&socket.TransportSSE{
			Cors: &socket.CorsConfig{
				MaxAge:              12 * time.Hour,
				AllowAllOrigins:     false,
				AllowCredentials:    true,
				AllowPrivateNetwork: false,
				AllowOrigins:        []string{"*"},
				AllowMethods:        []string{"GET", "POST", "OPTIONS"},
				AllowHeaders:        []string{"Origin", "Content-Length", "Content-Type"},
				ExposeHeaders:       []string{},
			},
		}},
	}

	// configure socket server
	router := chain.New()
	router.Configure("/socket", handler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		log.Printf("Socket listening on :%s...\n", port)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Socket HTTP server error: %v", err)
		}
	}()

	cluster[lastPortUsed] = server

	if ctx != nil {
		ctx.OK()
	}
}

func createStaticFileHandler() http.Handler {
	h := http.FileServer(http.FS(staticFs))
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
