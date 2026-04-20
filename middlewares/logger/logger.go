// Package logger provides structured logging middleware for Chain.
//
// The logging middleware uses Go's standard log/slog package and provides:
// - Request/response logging with status codes
// - Duration tracking for each request
// - Custom log formats
// - Request ID generation and tracking
// - Skip conditions for health checks and static assets
//
// Basic usage:
//
//	router.Use(logger.New())
//
// With custom configuration:
//
//	router.Use(logger.New(logger.Config{
//	    Format: logger.Format("%{method} %{path} %{status} %{latency}"),
//	    SkipPaths: []string{"/health", "/ping"},
//	}))
//
// With request ID:
//
//	router.Use(logger.New(logger.Config{
//	    RequestIDHeader: "X-Request-ID",
//	}))
package logger

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nidorx/chain"
)

// Format defines the log format placeholders available in the middleware.
type Format string

const (
	// FormatDefault is the default log format: [method path status latency]
	FormatDefault Format = "default"

	// FormatCombined is the Apache combined log format
	FormatCombined Format = "combined"

	// FormatJSON logs as structured JSON (uses slog's JSON handler)
	FormatJSON Format = "json"

	// FormatCustom allows custom format strings
	FormatCustom Format = "custom"
)

// Config holds the configuration for the logging middleware.
type Config struct {
	// Format specifies the log format (default, combined, json, custom).
	// Default: FormatDefault
	Format Format

	// CustomFormat is the custom format string when Format is FormatCustom.
	// Available placeholders:
	//   %{method}   - HTTP method
	//   %{path}     - Request path
	//   %{status}   - Response status code
	//   %{latency}  - Request duration
	//   %{ip}       - Client IP address
	//   %{useragent} - User-Agent header
	//   %{referer}  - Referer header
	//   %{host}     - Request host
	//   %{proto}    - HTTP protocol version
	//   %{reqid}    - Request ID
	//   %{err}      - Error message (if any)
	//   %{query}    - Query string
	CustomFormat string

	// Logger is the slog logger instance. If nil, uses slog.Default().
	Logger *slog.Logger

	// SkipPaths is a list of paths that should not be logged.
	// Useful for health check endpoints and static assets.
	SkipPaths []string

	// SkipPathPrefixes is a list of path prefixes that should not be logged.
	SkipPathPrefixes []string

	// RequestIDHeader is the header name to use for request IDs.
	// If the header is not present, a new KSUID will be generated.
	// Default: "X-Request-ID"
	RequestIDHeader string

	// GenerateRequestID, if true, generates a new request ID for each request
	// even if the header is not present.
	// Default: true
	GenerateRequestID bool

	// LogLevel allows setting the log level for different status code ranges.
	// Default: 2xx-3xx = Info, 4xx = Warn, 5xx = Error
	StatusLevelFunc func(status int) slog.Level

	// LatencyThreshold logs a warning if the request takes longer than this.
	LatencyThreshold time.Duration
}

// DefaultConfig returns a default configuration for the logging middleware.
func DefaultConfig() Config {
	return Config{
		Format:            FormatDefault,
		Logger:            slog.Default(),
		RequestIDHeader:   "X-Request-ID",
		GenerateRequestID: true,
	}
}

// RequestIDKey is the context key for storing the request ID.
const RequestIDKey = "chain.logger.request-id"

// New creates a logging middleware with the given configuration.
//
// Example:
//
//	// Default logging
//	router.Use(logger.New())
//
//	// Custom format
//	router.Use(logger.New(logger.Config{
//	    Format: logger.FormatCustom,
//	    CustomFormat: "%{method} %{path} took %{latency}",
//	}))
//
//	// Skip health checks
//	router.Use(logger.New(logger.Config{
//	    SkipPaths: []string{"/health", "/ping"},
//	}))
func New(config ...Config) chain.MiddlewareFunc {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultConfig()
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.RequestIDHeader == "" {
		cfg.RequestIDHeader = "X-Request-ID"
	}
	if !cfg.GenerateRequestID {
		cfg.GenerateRequestID = true
	}

	// Build skip path map for fast lookup
	skipPaths := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(ctx *chain.Context, next func() error) error {
		// Check if we should skip logging for this path
		path := ctx.URL().Path
		if skipPaths[path] {
			return next()
		}
		for _, prefix := range cfg.SkipPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				return next()
			}
		}

		// Get or generate request ID
		requestID := ctx.GetHeader(cfg.RequestIDHeader)
		if requestID == "" && cfg.GenerateRequestID {
			requestID = chain.NewUID()
		}
		if requestID != "" {
			ctx.Set(RequestIDKey, requestID)
		}

		// Track start time
		start := time.Now()

		// Execute the next handler
		err := next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := ctx.GetStatus()

		// Determine log level
		level := determineLogLevel(status, cfg.StatusLevelFunc)

		// Check latency threshold
		if cfg.LatencyThreshold > 0 && latency > cfg.LatencyThreshold {
			cfg.Logger.Warn(
				"slow request",
				slog.String("path", path),
				slog.Duration("latency", latency),
				slog.Duration("threshold", cfg.LatencyThreshold),
			)
		}

		// Format the log message
		var attrs []slog.Attr
		attrs = append(attrs,
			slog.String("method", ctx.Method()),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
		)

		if requestID != "" {
			attrs = append(attrs, slog.String("request_id", requestID))
		}
		if ip := ctx.Ip(); ip != "" {
			attrs = append(attrs, slog.String("ip", ip))
		}
		if ua := ctx.UserAgent(); ua != "" {
			attrs = append(attrs, slog.String("user_agent", ua))
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		// Log based on format type
		switch cfg.Format {
		case FormatJSON:
			cfg.Logger.LogAttrs(nil, level, "HTTP Request", attrs...)
		case FormatCombined:
			msg := fmt.Sprintf(`%s - - [%s] "%s %s %s" %d %d`,
				ctx.Ip(),
				start.Format("02/Jan/2006:15:04:05 -0700"),
				ctx.Method(),
				path,
				ctx.Request.Proto,
				status,
				0, // response size - would need ResponseWriterSpy for this
			)
			cfg.Logger.Log(nil, level, msg)
		case FormatCustom:
			msg := formatCustom(cfg.CustomFormat, ctx, status, latency, requestID, err)
			cfg.Logger.Log(nil, level, msg)
		default:
			cfg.Logger.Log(nil, level, fmt.Sprintf(
				"[%d] %s %s — %v",
				status, ctx.Method(), path, latency,
			))
		}

		return err
	}
}

// determineLogLevel returns the appropriate slog level based on status code.
func determineLogLevel(status int, levelFunc func(int) slog.Level) slog.Level {
	if levelFunc != nil {
		return levelFunc(status)
	}

	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// formatCustom formats a custom log message using placeholders.
func formatCustom(format string, ctx *chain.Context, status int, latency time.Duration, requestID string, err error) string {
	replacements := []struct {
		placeholder string
		value       string
	}{
		{"%{method}", ctx.Method()},
		{"%{path}", ctx.URL().Path},
		{"%{status}", fmt.Sprintf("%d", status)},
		{"%{latency}", latency.String()},
		{"%{ip}", ctx.Ip()},
		{"%{useragent}", ctx.UserAgent()},
		{"%{referer}", ctx.GetHeader("Referer")},
		{"%{host}", ctx.Host()},
		{"%{proto}", ctx.Request.Proto},
		{"%{reqid}", requestID},
		{"%{err}", func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}()},
		{"%{query}", ctx.URL().RawQuery},
	}

	for _, r := range replacements {
		format = strings.ReplaceAll(format, r.placeholder, r.value)
	}
	return format
}

// GetRequestID retrieves the request ID from the context.
// Returns empty string if no request ID was set.
func GetRequestID(ctx *chain.Context) string {
	if id, ok := ctx.Get(RequestIDKey); ok {
		if s, is := id.(string); is {
			return s
		}
	}
	return ""
}
