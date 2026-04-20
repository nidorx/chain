package cors

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/nidorx/chain"
)

var (
	// DefaultSchemas lists the default allowed origin schemas.
	DefaultSchemas = []string{
		"http://",
		"https://",
	}
	// ExtensionSchemas lists browser extension schemas.
	ExtensionSchemas = []string{
		"chrome-extension://",
		"safari-extension://",
		"moz-extension://",
		"ms-browser-extension://",
	}
	// FileSchemas lists file schemas.
	FileSchemas = []string{
		"file://",
	}
	// WebSocketSchemas lists WebSocket schemas.
	WebSocketSchemas = []string{
		"ws://",
		"wss://",
	}
)

type cors struct {
	allowAllOrigins            bool
	allowCredentials           bool
	allowOriginFunc            func(string) bool
	allowOriginWithContextFunc func(*chain.Context, string) bool
	allowOrigins               []string
	normalHeaders              http.Header
	preflightHeaders           http.Header
	wildcardOrigins            [][]string
	optionsResponseStatusCode  int
}

func newCors(config Config) *cors {
	if err := config.Validate(); err != nil {
		panic(err.Error())
	}

	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			config.AllowAllOrigins = true
		}
	}

	if config.OptionsResponseStatusCode == 0 {
		config.OptionsResponseStatusCode = http.StatusNoContent
	}

	return &cors{
		allowOriginFunc:            config.AllowOriginFunc,
		allowOriginWithContextFunc: config.AllowOriginWithContextFunc,
		allowAllOrigins:            config.AllowAllOrigins,
		allowCredentials:           config.AllowCredentials,
		allowOrigins:               normalize(config.AllowOrigins),
		normalHeaders:              generateNormalHeaders(config),
		preflightHeaders:           generatePreflightHeaders(config),
		wildcardOrigins:            config.parseWildcardRules(),
		optionsResponseStatusCode:  config.OptionsResponseStatusCode,
	}
}

func (c *cors) applyCors(ctx *chain.Context) {
	origin := ctx.Request.Header.Get("Origin")
	if len(origin) == 0 {
		// Request is not a CORS request
		return
	}
	host := ctx.Request.Host

	if origin == "http://"+host || origin == "https://"+host {
		// Request is not a CORS request but has origin header (e.g., fetch API)
		return
	}

	if !c.isOriginValid(ctx, origin) {
		ctx.Forbidden()
		return
	}

	if ctx.Request.Method == http.MethodOptions {
		c.handlePreflight(ctx)
		defer ctx.Status(c.optionsResponseStatusCode)
	} else {
		c.handleNormal(ctx)
	}

	if !c.allowAllOrigins {
		ctx.AddHeader("Access-Control-Allow-Origin", origin)
	}
}

func (c *cors) validateWildcardOrigin(origin string) bool {
	for _, w := range c.wildcardOrigins {
		if w[0] == "*" && strings.HasSuffix(origin, w[1]) {
			return true
		}
		if w[1] == "*" && strings.HasPrefix(origin, w[0]) {
			return true
		}
		if strings.HasPrefix(origin, w[0]) && strings.HasSuffix(origin, w[1]) {
			return true
		}
	}

	return false
}

func (c *cors) isOriginValid(ctx *chain.Context, origin string) bool {
	valid := c.validateOrigin(origin)
	if !valid && c.allowOriginWithContextFunc != nil {
		valid = c.allowOriginWithContextFunc(ctx, origin)
	}
	return valid
}

var originRegex = regexp.MustCompile(`^/(.+)/[gimuy]?$`)

func (c *cors) validateOrigin(origin string) bool {
	if c.allowAllOrigins {
		return true
	}

	for _, value := range c.allowOrigins {
		if !originRegex.MatchString(value) && value == origin {
			return true
		}

		if originRegex.MatchString(value) &&
			regexp.MustCompile(originRegex.FindStringSubmatch(value)[1]).MatchString(origin) {
			return true
		}
	}

	if len(c.wildcardOrigins) > 0 && c.validateWildcardOrigin(origin) {
		return true
	}

	if c.allowOriginFunc != nil {
		return c.allowOriginFunc(origin)
	}

	return false
}

func (c *cors) handlePreflight(ctx *chain.Context) {
	header := ctx.Writer.Header()
	for key, value := range c.preflightHeaders {
		header[key] = value
	}
}

func (c *cors) handleNormal(ctx *chain.Context) {
	header := ctx.Writer.Header()
	for key, value := range c.normalHeaders {
		header[key] = value
	}
}
