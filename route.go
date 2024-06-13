package chain

import (
	"fmt"
	"log/slog"
	"strings"
)

const (
	separator = '/'
	parameter = ':'
	wildcard  = '*'
)

type RouteConfigurator interface {
	Configure(router *Router, path string)
}

type MiddlewareHandler interface {
	Handle(ctx *Context, next func() error) error
}

type MiddlewareWithInitHandler interface {
	Init(method string, path string, router *Router)
	Handle(ctx *Context, next func() error) error
}

type Handle func(*Context) error

type Middleware struct {
	Path   *PathDetails
	Handle func(ctx *Context, next func() error) error
}

// Route control of a registered route
type Route struct {
	Path             *PathDetails
	Handle           Handle
	Middlewares      []*Middleware
	middlewaresAdded map[*Middleware]bool
}

// Dispatch ctx into this route
func (r *Route) Dispatch(ctx *Context) error {
	if len(r.Middlewares) == 0 {
		return r.Handle(ctx)
	}

	index := 0
	var next func() error
	next = func() error {
		if index > len(r.Middlewares)-1 {
			// end of middlewares
			return r.Handle(ctx)
		}

		middleware := r.Middlewares[index]
		index++

		match, names, values := middleware.Path.Match(ctx)
		if match {
			var nextErr error
			calledNext := false
			nextMid := func() error {
				if calledNext {
					slog.Warn(
						"[chain] calling next() multiple times for route",
						slog.Int("index", index),
						slog.String("path", ctx.path),
					)

					return nextErr
				}
				calledNext = true
				nextErr = next()
				return nextErr
			}

			if len(names) > 0 {
				// middleware expects parameterizable route
				return middleware.Handle(ctx.WithParams(names, values), nextMid)
			} else {
				// use same context
				return middleware.Handle(ctx, nextMid)
			}
		}
		return next()
	}
	return next()
}

// PathDetails represents all useful information about a dynamic path (used by handlers)
type PathDetails struct {
	path         string   // a rota original
	priority     int      // calculo da prioridade desse path
	segments     []string // Os segmentos desse path. Parametros são representados como ":" e wildcard como "*"
	hasStatic    bool     // possui estático
	hasParameter bool     // possui parametros
	hasWildcard  bool     // possui wildcard
	params       []string // os nomes dos parametros no path. Ex. ["category", "filepath"]
	paramsIndex  []int    // os indices de segmentos parametricos no path. Ex. [0, 2]
}

func (d PathDetails) Params() (segments []string, params []string, indexes []int) {
	return d.segments, d.params, d.paramsIndex
}

func (d PathDetails) String() string {
	return fmt.Sprintf(
		`PathDetails{path: "%v", hasStatic: %v, hasParameter: %v, hasWildcard: %v, params: [%v], priority: %v, segments: [%v]}`,
		d.path, d.hasStatic, d.hasParameter, d.hasWildcard, strings.Join(d.params, ", "), d.priority, strings.Join(d.segments, ", "),
	)
}

func (d PathDetails) ReplacePath(ctx *Context) string {
	const stackBufSize = 128

	// Use a static sized buffer on the stack in the common case.
	// If the path is too long, allocate a buffer on the heap instead.
	buf := make([]byte, 0, stackBufSize)
	if l := len(ctx.path) + 1; l > stackBufSize {
		buf = make([]byte, 0, l)
	}

	for j, segment := range d.segments {
		buf = append(buf, '/')
		if strings.IndexByte(segment, parameter) == 0 {
			buf = append(buf, []byte(ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]])...)
			continue
		}

		if strings.IndexByte(segment, wildcard) == 0 {
			buf = append(buf, []byte(ctx.path[ctx.pathSegments[j]+1:])...)
			break
		}

		buf = append(buf, []byte(segment)...)
	}

	return string(buf)
}

func (d PathDetails) FastMatch(ctx *Context) bool {
	if !d.hasWildcard && len(d.segments) < ctx.pathSegmentsCount {
		return false
	}

	if len(d.segments) > ctx.pathSegmentsCount {
		return false
	}

	for j, segment := range d.segments {
		if strings.IndexByte(segment, parameter) == 0 {
			continue
		}

		if strings.IndexByte(segment, wildcard) == 0 {
			return true
		}

		if segment != ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]] {
			return false
		}
	}
	return true
}

// Match checks if the patch is compatible, performs the extraction of parameters
func (d PathDetails) Match(ctx *Context) (match bool, paramNames []string, paramValues []string) {
	if d.FastMatch(ctx) {
		match = true
		paramNames = d.params
		for _, index := range d.paramsIndex {
			if strings.IndexByte(d.segments[index], wildcard) == 0 {
				paramValues = append(paramValues, ctx.path[ctx.pathSegments[index]:])
				break
			}
			paramValues = append(paramValues, ctx.path[ctx.pathSegments[index]+1:ctx.pathSegments[index+1]])
		}
	} else {
		paramNames = nil
		paramValues = nil
	}

	return
}

// MaybeMatches checks if this path is applicable over the other. Used for registering middlewares in routes
func (d PathDetails) MaybeMatches(o *PathDetails) bool {

	if d.path == o.path {
		return true
	}

	if len(d.segments) > len(o.segments) {
		if !o.hasWildcard {
			return false
		}

		for j, oSegment := range o.segments {
			if oSegment == "*" {
				return true
			}
			iSegment := d.segments[j]
			switch iSegment {
			case string(parameter):
				//  this: /blog/:|category/:page/:subpage
				// other: /blog/category/*filepath | /blog/:category/*filepath | /blog/*filepath
				continue
			default:
				if oSegment != ":" && oSegment != iSegment {
					//  this: /blog/[>>category-1<<]/:page/:subpage
					// other: /blog/category-2/*filepath
					return false
				}
			}
		}

		return false
	}

	if !d.hasWildcard && len(d.segments) < len(o.segments) {
		return false
	}

	for j, iSegment := range d.segments {
		switch iSegment {
		case string(parameter):
			//  this: /blog/:|category/*page
			// other: /blog/category/:page | /blog/category/page | /blog/category/page/subpage
			continue
		case string(wildcard):
			//  this: /blog/category/*page
			// other: /blog/category/:page | /blog/category/page | /blog/category/page/subpage
			return true
		default:
			oSegment := o.segments[j]
			if oSegment == ":" {
				continue
			}
			if iSegment != oSegment {
				//  this: /blog/[>>category-2<<]/*page
				// other: /blog/category-1/:page
				return false
			}
		}
	}

	return true
}

func (d PathDetails) conflictsWith(o *PathDetails) bool {
	if d.priority != o.priority {
		return false
	}

	for j, iSegment := range d.segments {
		oSegment := o.segments[j]
		//if iSegment == "*" && iSegment == oSegment {
		//	return true
		//}
		if iSegment != oSegment {
			return false
		}
	}
	return true
}

// ParsePathDetails obtém informações sobre um path dinamico.
func ParsePathDetails(pathOrig string) *PathDetails {

	// uses a path with at the beginning and end to facilitate the loop (details.segments++ rule)
	if !strings.HasPrefix(pathOrig, string(separator)) {
		pathOrig = string(separator) + pathOrig
	}

	details := &PathDetails{
		path: pathOrig,
	}

	staticLength := 0

	ctx := &Context{path: pathOrig}
	ctx.parsePathSegments() // reuse path segments logic
	path := pathOrig[0:]

	for i := 0; i < ctx.pathSegmentsCount; i++ {
		part := path[ctx.pathSegments[i]+1 : ctx.pathSegments[i+1]]
		if strings.IndexByte(part, parameter) == 0 {
			if len(part) == 1 {
				panic(fmt.Sprintf("[chain] is necessary to inform the name of the parameter. path: %s", path))
			}
			paramName := part[1:]
			if strings.IndexByte(paramName, wildcard) >= 0 || strings.IndexByte(paramName, parameter) >= 0 {
				panic(fmt.Sprintf("[chain] only one wildcard per path segment is allowed. path: %s", path))
			}
			details.hasParameter = true
			details.segments = append(details.segments, string(parameter))
			details.params = append(details.params, paramName)
			details.paramsIndex = append(details.paramsIndex, i)
		} else if strings.IndexByte(part, wildcard) == 0 {
			if details.hasWildcard {
				panic(fmt.Sprintf("[chain] catch-all routes are only allowed at the end of the path. path: %s", path))
			}
			paramName := part[1:]
			if paramName == "" {
				paramName = "filepath"
			}
			if strings.IndexByte(paramName, wildcard) >= 0 || strings.IndexByte(paramName, parameter) >= 0 {
				panic(fmt.Sprintf("[chain] only one wildcard per path segment is allowed. path: %s", path))
			}
			details.hasWildcard = true
			details.segments = append(details.segments, string(wildcard))
			details.params = append(details.params, paramName)
			details.paramsIndex = append(details.paramsIndex, i)
		} else {
			details.hasStatic = true
			staticLength = staticLength + len(part)
			details.segments = append(details.segments, part)
		}
	}

	// Calculating the priority of this handler
	//
	// a) Left parts have higher priority than right
	// b) For each part of the path
	//    1. ("*") Catch all parameter has weight 1
	//    2. (":") Named parameter has weight 2
	//    3. (".") An exact match has weight 3
	for i, segment := range details.segments {
		weight := 3
		if strings.IndexByte(segment, parameter) == 0 {
			weight = 2
		} else if strings.IndexByte(segment, wildcard) == 0 {
			weight = 1
		}
		height := ctx.pathSegmentsCount - i
		details.priority = details.priority + (height * height * weight)
	}

	return details
}
