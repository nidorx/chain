package cors

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nidorx/chain"
)

// Config represents all available options for the CORS middleware.
type Config struct {
	// AllowAllOrigins, if true, allows all origins.
	// Conflicts with AllowOrigins, AllowOriginFunc, and AllowOriginWithContextFunc.
	AllowAllOrigins bool

	// AllowOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is [].
	AllowOrigins []string

	// AllowOriginFunc is a custom function to validate the origin. It takes the origin
	// as an argument and returns true if allowed or false otherwise. If this option is
	// set, the content of AllowOrigins is ignored.
	AllowOriginFunc func(origin string) bool

	// AllowOriginWithContextFunc is the same as AllowOriginFunc except it also receives
	// the full request context. This function should use the context as a read-only source
	// and not have any side effects on the request, such as aborting or injecting values.
	AllowOriginWithContextFunc func(c *chain.Context, origin string) bool

	// AllowMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS).
	AllowMethods []string

	// AllowPrivateNetwork indicates whether the response should include the
	// Access-Control-Allow-Private-Network header.
	AllowPrivateNetwork bool

	// AllowHeaders is a list of non-simple headers the client is allowed to use with
	// cross-domain requests.
	AllowHeaders []string

	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool

	// ExposeHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification.
	ExposeHeaders []string

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached. Default is 43200 (12 hours).
	MaxAge time.Duration

	// AllowWildcard allows usage of wildcard patterns in AllowOrigins like
	// http://some-domain/*, https://api.* or http://some.*.subdomain.com.
	AllowWildcard bool

	// AllowBrowserExtensions allows usage of popular browser extensions schemas
	// (chrome-extension://, safari-extension://, moz-extension://, ms-browser-extension://).
	AllowBrowserExtensions bool

	// CustomSchemas allows adding custom schema like tauri://.
	CustomSchemas []string

	// AllowWebSockets allows usage of WebSocket protocol (ws://, wss://).
	AllowWebSockets bool

	// AllowFiles allows usage of file:// schema (dangerous! use only when 100% sure it's needed).
	AllowFiles bool

	// OptionsResponseStatusCode allows setting a custom OPTIONS response status code
	// for old browsers/clients. Default is 204 (No Content).
	OptionsResponseStatusCode int
}

// AddAllowMethods adds custom methods to the allowed methods list.
func (c *Config) AddAllowMethods(methods ...string) {
	c.AllowMethods = append(c.AllowMethods, methods...)
}

// AddAllowHeaders adds custom headers to the allowed headers list.
func (c *Config) AddAllowHeaders(headers ...string) {
	c.AllowHeaders = append(c.AllowHeaders, headers...)
}

// AddExposeHeaders adds custom headers to the expose headers list.
func (c *Config) AddExposeHeaders(headers ...string) {
	c.ExposeHeaders = append(c.ExposeHeaders, headers...)
}

func (c Config) getAllowedSchemas() []string {
	allowedSchemas := DefaultSchemas
	if c.AllowBrowserExtensions {
		allowedSchemas = append(allowedSchemas, ExtensionSchemas...)
	}
	if c.AllowWebSockets {
		allowedSchemas = append(allowedSchemas, WebSocketSchemas...)
	}
	if c.AllowFiles {
		allowedSchemas = append(allowedSchemas, FileSchemas...)
	}
	if c.CustomSchemas != nil {
		allowedSchemas = append(allowedSchemas, c.CustomSchemas...)
	}
	return allowedSchemas
}

var regexpBasedOrigin = regexp.MustCompile(`^\/(.+)\/[gimuy]?$`)

func (c Config) validateAllowedSchemas(origin string) bool {
	allowedSchemas := c.getAllowedSchemas()

	if regexpBasedOrigin.MatchString(origin) {
		// Normalize regexp-based origins
		origin = regexpBasedOrigin.FindStringSubmatch(origin)[1]
		origin = strings.Replace(origin, "?", "", 1)

		// Strip leading ^ anchor for schema validation
		// The anchor is part of regex syntax, not the URL scheme
		origin = strings.TrimPrefix(origin, "^")
	}

	for _, schema := range allowedSchemas {
		if strings.HasPrefix(origin, schema) {
			return true
		}
	}
	return false
}

// Validate checks the configuration for conflicts and errors.
func (c Config) Validate() error {
	hasOriginFn := c.AllowOriginFunc != nil || c.AllowOriginWithContextFunc != nil

	if c.AllowAllOrigins && (hasOriginFn || len(c.AllowOrigins) > 0) {
		originFields := strings.Join([]string{
			"AllowOriginFunc",
			"AllowOriginWithContextFunc",
			"AllowOrigins",
		}, " or ")
		return fmt.Errorf(
			"cors: conflict settings: all origins enabled. %s is not needed",
			originFields,
		)
	}
	if !c.AllowAllOrigins && !hasOriginFn && len(c.AllowOrigins) == 0 {
		return errors.New("cors: conflict settings: all origins disabled")
	}
	for _, origin := range c.AllowOrigins {
		if !strings.Contains(origin, "*") && !c.validateAllowedSchemas(origin) {
			return errors.New("cors: bad origin: origins must contain '*' or include " + strings.Join(c.getAllowedSchemas(), ","))
		}
	}
	return nil
}

func (c Config) parseWildcardRules() [][]string {
	var wRules [][]string

	if !c.AllowWildcard {
		return wRules
	}

	for _, o := range c.AllowOrigins {
		if !strings.Contains(o, "*") {
			continue
		}

		if c := strings.Count(o, "*"); c > 1 {
			panic("cors: only one * is allowed in origin")
		}

		i := strings.Index(o, "*")
		if i == 0 {
			wRules = append(wRules, []string{"*", o[1:]})
			continue
		}
		if i == (len(o) - 1) {
			wRules = append(wRules, []string{o[:i], "*"})
			continue
		}

		wRules = append(wRules, []string{o[:i], o[i+1:]})
	}

	return wRules
}

// DefaultConfig returns a generic default configuration.
func DefaultConfig() Config {
	return Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// Default returns the CORS middleware with default configuration (all origins allowed).
func Default() chain.Handle {
	config := DefaultConfig()
	config.AllowAllOrigins = true
	return New(config)
}

// New returns the CORS middleware with user-defined custom configuration.
func New(config Config) chain.Handle {
	c := newCors(config)
	return func(ctx *chain.Context) error {
		c.applyCors(ctx)
		return nil
	}
}
