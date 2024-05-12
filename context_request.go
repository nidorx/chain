package chain

import (
	"io"
	"net/http"
	"net/url"
)

// BodyAs(obj) err                     // request body as specified class (deserialized from JSON)
// QueryAs(obj) err                    // request body as specified class (deserialized from JSON)

// Request methods
// body()                                // request body as string
// bodyAsBytes()                         // request body as array of bytes

// BodyBytes get body as array of bytes
func (ctx *Context) BodyBytes() (body []byte, err error) {
	if cb := ctx.Get(BodyBytesKey); cb != nil {
		if cbb, ok := cb.([]byte); ok {
			body = cbb
		}
	}

	if body == nil {
		body, err = io.ReadAll(ctx.Request.Body)
		if err != nil {
			return
		}
		ctx.Set(BodyBytesKey, body)
	}

	return
}

// bodyStreamAsClass(clazz)              // request body as specified class (memory optimized version of above)
// bodyValidator(clazz)                  // request body as validator typed as specified class
// bodyInputStream()                     // the underyling input stream of the request

// formParam("name")                     // form parameter by name, as string
// formParamAsClass("name", clazz)       // f orm parameter by name, as validator typed as specified class
// formParams("name")                    // list of form parameters by name
// formParamMap()                        // map of all form parameters

// pathParam("name")                     // path parameter by name as string
// pathParamAsClass("name", clazz)       // path parameter as validator typed as specified class
// pathParamMap()                        // map of all path parameters

// queryParam("name")                    // query param by name as string
// queryParamAsClass("name", clazz)      // query param parameter by name, as validator typed as specified class
// queryParams("name")                   // list of query parameters by name
// queryParamMap()                       // map of all query parameters
// queryString()                         // full query string

// uploadedFile("name")                  // uploaded file by name
// uploadedFiles("name")                 // all uploaded files by name
// uploadedFiles()                       // all uploaded files as list
// uploadedFileMap()                     // all uploaded files as a "names by files" map

// basicAuthCredentials()                // basic auth credentials (or null if not set)

// attribute("name", value)              // set an attribute on the request
// attribute("name")                     // get an attribute on the request
// attributeOrCompute("name", ctx -> {}) // get an attribute or compute it based on the context if absent
// attributeMap()                        // map of all attributes on the request

// contentLength()                       // content length of the request body
// contentType()                         // request content type

// isMultipart()                         // true if the request is multipart
// isMultipartFormData()                 // true if the request is multipart/formdata

// sessionAttribute("name", value)       // set a session attribute
// sessionAttribute("name")              // get a session attribute
// consumeSessionAttribute("name")       // get a session attribute, and set value to null
// cachedSessionAttribute("name", value) // set a session attribute, and cache the value as a request attribute
// cachedSessionAttribute("name")        // get a session attribute, and cache the value as a request attribute
// cachedSessionAttributeOrCompute(...)  // same as above, but compute and set if value is absent
// sessionAttributeMap()                 // map of all session attributes
// cookieMap()                           // map of all request cookies

// Host host as string
func (ctx *Context) Host() string {
	return ctx.Request.Host
}

// Host ip as string
func (ctx *Context) Ip() string {
	return ctx.Request.RemoteAddr
}

// Method specifies the HTTP method (GET, POST, PUT, etc.).
func (ctx *Context) Method() string {
	return ctx.Request.Method
}

// UserAgent returns the client's User-Agent, if sent in the request.
func (ctx *Context) UserAgent() string {
	return ctx.Request.UserAgent()
}

// URL request url
func (ctx *Context) URL() *url.URL {
	return ctx.Request.URL
}

// GetContentType returns the Content-Type header of the request.
func (ctx *Context) GetContentType() string {
	return filterFlags(ctx.Request.Header.Get("Content-Type"))
}

// GetCookie returns the named cookie provided in the request or nil if not found.
// If multiple cookies match the given name, only one cookie will be returned.
func (ctx *Context) GetCookie(name string) *http.Cookie {
	// @todo: ctx.Request.readCookies is slow
	if cookie, err := ctx.Request.Cookie(name); err == nil {
		return cookie
	}
	return nil
}

// header("name")                        // request header by name (can be used with Header.HEADERNAME)
// headerAsClass("name", clazz)          // request header by name, as validator typed as specified class
// headerMap()                           // map of all request headers

// GetHeader gets the first value associated with the given key. If there are no values associated with the key,
// GetHeader returns "".
// It is case insensitive; textproto.CanonicalMIMEHeaderKey is used to canonicalize the provided key. Get assumes
// that all keys are stored in canonical form. To use non-canonical keys, access the map directly.
func (ctx *Context) GetHeader(key string) string {
	return ctx.Writer.Header().Get(key)
}

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}
