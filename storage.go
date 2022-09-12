package chain

import (
	"sort"
	"strings"
)

type RouteStorage struct {
	routes map[int][]*Route // by num of segments
}

func (s *RouteStorage) add(route *Route) {
	details := route.Path

	numSegments := len(details.segments)
	if s.routes == nil {
		s.routes = map[int][]*Route{}
	}
	if s.routes[numSegments] == nil {
		s.routes[numSegments] = []*Route{}

		// inserts wildcards from lower levels into this list
		for oNumSegments, routes := range s.routes {
			if oNumSegments < numSegments {
				for _, other := range routes {
					if other.Path.hasWildcard {
						s.routes[numSegments] = append(s.routes[numSegments], other)
					}
				}
			}
		}
	}
	s.routes[numSegments] = append(s.routes[numSegments], route)

	sort.Slice(s.routes[numSegments], func(i, j int) bool {
		// high priority at the beginning'
		return s.routes[numSegments][i].Path.priority > s.routes[numSegments][j].Path.priority
	})

	if details.hasWildcard {
		// inserts this new path in the upper segments and does the reordering
		for oNumSegments, _ := range s.routes {
			if oNumSegments > numSegments {
				s.routes[oNumSegments] = append(s.routes[oNumSegments], route)
				sort.Slice(s.routes[oNumSegments], func(i, j int) bool {
					return s.routes[oNumSegments][i].Path.priority > s.routes[oNumSegments][j].Path.priority
				})
			}
		}
	}
}

func (s *RouteStorage) lookup(ctx *Context) *Route {

	if s.routes == nil {
		return nil
	}

	var (
		path          = ctx.path
		segments      = ctx.pathSegments
		segmentsCount = ctx.pathSegmentsCount
	)

	for i := segmentsCount; i > 0; i-- {
		routes := s.routes[i]
		if routes == nil {
			continue
		}

	nextRoute:
		for _, route := range routes {
			details := route.Path
			if !details.hasWildcard && i < segmentsCount {
				// at this point it's just looking for the wildcard that satisfies this route
				continue
			}

			// same effect as ` !details.FastMatch(ctx)`, but faster

			for j, segment := range details.segments {
				if strings.IndexByte(segment, wildcard) == 0 {
					break
				}

				if strings.IndexByte(segment, parameter) == 0 && ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]] != "" {
					continue
				}

				if segment != ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]] {
					continue nextRoute
				}
			}

			// found, populate parameters
			if details.hasWildcard {
				for j, index := range details.paramsIndex {
					if j == len(details.paramsIndex)-1 {
						ctx.addParameter(details.params[j], path[segments[index]:])
						break
					}
					ctx.addParameter(details.params[j], path[segments[index]+1:segments[index+1]])
				}
			} else {
				for j, index := range details.paramsIndex {
					ctx.addParameter(details.params[j], path[segments[index]+1:segments[index+1]])
				}
			}

			return route
		}

		// it only does the search in a single height
		break
	}

	return nil
}

func (s *RouteStorage) lookupCaseInsensitive(ctx *Context) *Route {

	if s.routes == nil {
		return nil
	}

	var (
		segmentsCount = ctx.pathSegmentsCount
	)

	for i := segmentsCount; i > 0; i-- {
		routes := s.routes[i]
		if routes == nil {
			continue
		}

	nextRoute:
		for _, route := range routes {
			details := route.Path
			if !details.hasWildcard && i < segmentsCount {
				// at this point it's just looking for the wildcard that satisfies this route
				continue
			}

			// same effect as ` !details.FastMatch(ctx)`, but faster

			for j, segment := range details.segments {
				if strings.IndexByte(segment, wildcard) == 0 {
					break
				}

				if strings.IndexByte(segment, parameter) == 0 && ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]] != "" {
					continue
				}

				if !strings.EqualFold(segment, ctx.path[ctx.pathSegments[j]+1:ctx.pathSegments[j+1]]) {
					continue nextRoute
				}
			}

			return route
		}

		// it only does the search in a single height
		break
	}

	return nil
}
