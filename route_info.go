package chain

import (
	"bytes"
	"fmt"
	"strings"
	"unsafe"
)

// RouteInfo represents all useful information about a dynamic path (used by handlers)
type RouteInfo struct {
	path         string   // a rota original
	pattern      string   // a rota, sem os nomes de parametros
	priority     int      // calculo da prioridade desse path
	hasStatic    bool     // possui estático
	hasParameter bool     // possui parametros
	hasWildcard  bool     // possui wildcard
	segments     []string // Os segmentos desse path. Parametros são representados como ":" e wildcard como "*"
	params       []string // os nomes dos parametros no path. Ex. ["category", "filepath"]
	paramsIndex  []int    // os indices de segmentos parametricos no path. Ex. [0, 2]
}

func (d *RouteInfo) Path() string {
	return d.path
}

func (d *RouteInfo) Pattern() string {
	return d.pattern
}

func (d *RouteInfo) Priority() int {
	return d.priority
}

func (d *RouteInfo) Params() []string {
	return d.params
}

func (d *RouteInfo) ParamsIndex() []int {
	return d.paramsIndex
}

func (d *RouteInfo) Segments() []string {
	return d.segments
}

func (d *RouteInfo) Details() (segments []string, params []string, indexes []int) {
	return d.segments, d.params, d.paramsIndex
}

func (d *RouteInfo) HasStatic() bool {
	return d.hasStatic
}

func (d *RouteInfo) HasParameter() bool {
	return d.hasParameter
}

func (d *RouteInfo) HasWildcard() bool {
	return d.hasWildcard
}

func (d *RouteInfo) ReplacePath(ctx *Context) string {
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

	return unsafe.String(unsafe.SliceData(buf), len(buf))
}

func (d *RouteInfo) FastMatch(ctx *Context) bool {
	// `/route/to` vs `/the/page/requested`
	if !d.hasWildcard && len(d.segments) < ctx.pathSegmentsCount {
		return false
	}

	// `/route/to/page/:id` vs `/the/page/requested`
	if len(d.segments) > ctx.pathSegmentsCount {
		return false
	}

	for j, segment := range d.segments {
		if strings.IndexByte(segment, parameter) == 0 {
			continue
		}

		// `/assets/*` vs `/assets/js/chain.js`
		if strings.IndexByte(segment, wildcard) == 0 {
			return true
		}

		// `/assets/*` vs `/files/js/chain.js`
		if segment != ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]] {
			return false
		}
	}
	return true
}

// Match checks if the patch is compatible, performs the extraction of parameters
func (d RouteInfo) Match(ctx *Context) (match bool, paramNames []string, paramValues []string) {
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

// Matches checks if this path is applicable over the other. Used for registering middlewares in routes
func (d *RouteInfo) Matches(o *RouteInfo) bool {

	if d.path == o.path || d.pattern == o.pattern {
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

func (d RouteInfo) conflictsWith(o *RouteInfo) bool {
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

func (d *RouteInfo) String() string {
	return fmt.Sprintf(
		`RouteInfo{path: "%v", hasStatic: %v, hasParameter: %v, hasWildcard: %v, params: [%v], priority: %v, segments: [%v]}`,
		d.path, d.hasStatic, d.hasParameter, d.hasWildcard, strings.Join(d.params, ", "), d.priority, strings.Join(d.segments, ", "),
	)
}

// ParseRouteInfo obtém informações sobre um path dinamico.
func ParseRouteInfo(pathOrig string) *RouteInfo {

	// uses a path with at the beginning and end to facilitate the loop (details.segments++ rule)
	if !strings.HasPrefix(pathOrig, string(separator)) {
		pathOrig = string(separator) + pathOrig
	}

	var (
		path              = pathOrig[0:]
		details           = &RouteInfo{path: pathOrig}
		pathSegments      [32]int
		staticLength      = 0
		pathSegmentsCount = parsePathSegments(pathOrig, &pathSegments)
	)

	for i := 0; i < pathSegmentsCount; i++ {
		part := path[pathSegments[i]+1 : pathSegments[i+1]]
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

	route := bytes.NewBufferString("")

	// Calculating the priority of this handler
	//
	// a) Left parts have higher priority than right
	// b) For each part of the path
	//    1. ("*") Catch all parameter has weight 1
	//    2. (":") Named parameter has weight 2
	//    3. (".") An exact match has weight 3
	for i, segment := range details.segments {
		weight := 3
		route.WriteByte('/')
		if strings.IndexByte(segment, parameter) == 0 {
			weight = 2
			route.WriteRune(parameter)
		} else if strings.IndexByte(segment, wildcard) == 0 {
			weight = 1
			route.WriteRune(wildcard)
		} else {
			route.WriteString(segment)
		}
		height := pathSegmentsCount - i
		details.priority = details.priority + (height * height * weight)
	}

	details.pattern = route.String()

	return details
}

func parsePathSegments(path string, pathSegments *[32]int) (pathSegmentsCount int) {
	var (
		segmentStart = 0
		segmentSize  int
	)
	if strings.IndexByte(path, separator) == 0 {
		path = path[1:]
	}

	pathSegments[0] = 0
	pathSegmentsCount = 1

	for {
		segmentSize = strings.IndexByte(path, separator)
		if segmentSize == -1 {
			segmentSize = len(path)
		}
		pathSegments[pathSegmentsCount] = segmentStart + 1 + segmentSize

		if segmentSize == len(path) {
			break
		}
		pathSegmentsCount++
		path = path[segmentSize+1:]
		segmentStart = segmentStart + 1 + segmentSize
	}

	return
}
