package chain

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownConfig holds configuration for graceful shutdown behavior.
type ShutdownConfig struct {
	// Timeout is the maximum duration to wait for in-flight requests
	// to complete before forcefully shutting down.
	// If zero, a default of 30 seconds is used.
	Timeout time.Duration

	// Signals specifies which OS signals trigger graceful shutdown.
	// If nil, defaults to []os.Signal{syscall.SIGINT, syscall.SIGTERM}.
	Signals []os.Signal
}

// DefaultShutdownTimeout is the default timeout for graceful shutdown.
const DefaultShutdownTimeout = 30 * time.Second

// Server wraps an http.Server with graceful shutdown support.
//
// Example:
//
//	r := chain.New()
//	r.GET("/", handler)
//
//	server := chain.NewServer(r, ":8080")
//	if err := server.ListenAndServe(); err != nil {
//	    log.Fatal(err)
//	}
type Server struct {
	// Server is the underlying http.Server.
	Server *http.Server

	// Config holds the shutdown configuration.
	Config ShutdownConfig

	// onShutdown is called when shutdown begins, before waiting for in-flight requests.
	onShutdown func()

	// onStop is called after all in-flight requests have completed or timeout reached.
	onStop func()

	mu       sync.Mutex
	stopChan chan struct{}
	shutting bool
}

// NewServer creates a new Server with graceful shutdown support.
// The addr parameter is the same as http.Server.Addr (e.g., ":8080").
//
// Example:
//
//	r := chain.New()
//	server := chain.NewServer(r, ":8080")
//	server.ListenAndServe()
func NewServer(router *Router, addr string) *Server {
	return &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		Config: ShutdownConfig{
			Timeout: DefaultShutdownTimeout,
		},
	}
}

// NewServerWithConfig creates a new Server with custom configuration.
//
// Example:
//
//	r := chain.New()
//	server := chain.NewServerWithConfig(r, ":8080", chain.ShutdownConfig{
//	    Timeout: 60 * time.Second,
//	})
func NewServerWithConfig(router *Router, addr string, config ShutdownConfig) *Server {
	if config.Timeout <= 0 {
		config.Timeout = DefaultShutdownTimeout
	}
	return &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		Config: config,
	}
}

// ListenAndServe starts the server and blocks until the server is shut down.
// It listens on the configured address and handles graceful shutdown on OS signals.
//
// Example:
//
//	server := chain.NewServer(router, ":8080")
//	if err := server.ListenAndServe(); err != nil {
//	    log.Fatal(err)
//	}
func (s *Server) ListenAndServe() error {
	if s.Server.Addr == "" {
		s.Server.Addr = ":http"
	}

	// Start server in a goroutine
	go func() {
		if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Server error (not from graceful shutdown)
			select {
			case <-s.stopChan:
				// We're shutting down, ignore this error
			default:
				// Not shutting down, this is a real error
			}
		}
	}()

	// Wait for shutdown signal
	s.waitForShutdownSignal()

	// Perform graceful shutdown
	return s.Shutdown(context.Background())
}

// ListenAndServeTLS starts the server with TLS and blocks until the server is shut down.
//
// Example:
//
//	server := chain.NewServer(router, ":8443")
//	if err := server.ListenAndServeTLS("cert.pem", "key.pem"); err != nil {
//	    log.Fatal(err)
//	}
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	if s.Server.Addr == "" {
		s.Server.Addr = ":https"
	}

	// Start server in a goroutine
	go func() {
		if err := s.Server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			select {
			case <-s.stopChan:
			default:
			}
		}
	}()

	// Wait for shutdown signal
	s.waitForShutdownSignal()

	// Perform graceful shutdown
	return s.Shutdown(context.Background())
}

// Shutdown initiates a graceful shutdown with the provided context.
// If ctx is nil, a background context with the configured timeout is used.
//
// This method can be called directly for programmatic shutdown:
//
//	server.Shutdown(nil)
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.shutting {
		s.mu.Unlock()
		return nil // Already shutting down
	}
	s.shutting = true
	s.mu.Unlock()

	if s.stopChan == nil {
		s.stopChan = make(chan struct{})
	}

	// Call onShutdown hook if set
	if s.onShutdown != nil {
		s.onShutdown()
	}

	// Use configured timeout if no context provided
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), s.Config.Timeout)
		defer cancel()
	}

	// Shutdown the HTTP server (stops accepting new connections, waits for in-flight)
	if err := s.Server.Shutdown(ctx); err != nil {
		if s.onStop != nil {
			s.onStop()
		}
		close(s.stopChan)
		return err
	}

	// Call onStop hook if set
	if s.onStop != nil {
		s.onStop()
	}

	close(s.stopChan)
	return nil
}

// Stop signals the server to shut down gracefully.
// This is useful for triggering shutdown from application code.
func (s *Server) Stop() error {
	return s.Shutdown(nil)
}

// OnShutdown registers a callback that is called when shutdown begins,
// before waiting for in-flight requests to complete.
func (s *Server) OnShutdown(fn func()) {
	s.onShutdown = fn
}

// OnStop registers a callback that is called after all in-flight requests
// have completed or the shutdown timeout is reached.
func (s *Server) OnStop(fn func()) {
	s.onStop = fn
}

// waitForShutdownSignal blocks until a shutdown signal is received.
func (s *Server) waitForShutdownSignal() {
	signals := s.Config.Signals
	if signals == nil {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, signals...)

	// Block until signal
	<-stop

	// Unregister signal notification
	signal.Stop(stop)
}

// IsShuttingDown returns true if the server is in the process of shutting down.
func (s *Server) IsShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutting
}

// Wait blocks until the server has fully shut down.
// This is useful for coordinating cleanup after shutdown.
func (s *Server) Wait() {
	if s.stopChan == nil {
		return
	}
	<-s.stopChan
}

// GracefulMiddleware is a middleware that checks if the server is shutting down.
// If shutting down, it sets the "Connection: close" header to prevent
// new connections from being kept alive.
func GracefulMiddleware(server *Server) func(ctx *Context, next func() error) error {
	return func(ctx *Context, next func() error) error {
		if server.IsShuttingDown() {
			ctx.SetHeader("Connection", "close")
		}
		return next()
	}
}
