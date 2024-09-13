package chain

import (
	"reflect"
	"testing"
)

func Test_PathInfo_extract(t *testing.T) {
	routes := []struct {
		path string
		info string
	}{
		{
			path: "/",
			info: `RouteInfo{path: "/", hasStatic: true, hasParameter: false, hasWildcard: false, params: [], priority: 3, segments: []}`,
		},
		{
			path: "/doc/",
			info: `RouteInfo{path: "/doc/", hasStatic: true, hasParameter: false, hasWildcard: false, params: [], priority: 15, segments: [doc, ]}`,
		},
		{
			path: "/search/",
			info: `RouteInfo{path: "/search/", hasStatic: true, hasParameter: false, hasWildcard: false, params: [], priority: 15, segments: [search, ]}`,
		},
		{
			path: "/user/:name",
			info: `RouteInfo{path: "/user/:name", hasStatic: true, hasParameter: true, hasWildcard: false, params: [name], priority: 14, segments: [user, :]}`,
		},
		{
			path: "/search/:query",
			info: `RouteInfo{path: "/search/:query", hasStatic: true, hasParameter: true, hasWildcard: false, params: [query], priority: 14, segments: [search, :]}`,
		},
		{
			path: "/cmd/:tool/",
			info: `RouteInfo{path: "/cmd/:tool/", hasStatic: true, hasParameter: true, hasWildcard: false, params: [tool], priority: 38, segments: [cmd, :, ]}`,
		},
		{
			path: "/src/*filepath",
			info: `RouteInfo{path: "/src/*filepath", hasStatic: true, hasParameter: false, hasWildcard: true, params: [filepath], priority: 13, segments: [src, *]}`,
		},
		{
			path: "/user/:name/about",
			info: `RouteInfo{path: "/user/:name/about", hasStatic: true, hasParameter: true, hasWildcard: false, params: [name], priority: 38, segments: [user, :, about]}`,
		},
		//"/info/:user/public",
		//"/cmd/:tool/:sub"
		//"/files/:dir/*filepath",
		{
			path: "/files/:dir/*filepath",
			info: `RouteInfo{path: "/files/:dir/*filepath", hasStatic: true, hasParameter: true, hasWildcard: true, params: [dir, filepath], priority: 36, segments: [files, :, *]}`,
		},
		//"/src/js/:folder/:name/:file",
		//"/src/:type/vendors/:name/index",
		//"/src/css/:folder/:name/:file",
		//"/src/:type/c/:name/index",
		//"/info/:user/project/:project",
		//"/src/a/:folder/:name/:file",
		//"/src/:type/b/:name/index",
		//"/src/b/:folder/:name/:file",
		//"/src/c/:folder/:name/:file",
		//"/src/:type/a/:name/index",
		//"/src/d/:folder/:name/*file",
	}
	for _, tt := range routes {
		t.Run(tt.path, func(t *testing.T) {
			a := ParseRouteInfo(tt.path)
			//RouteInfo{
			//	path: "/doc/", segments: 1, hasStatic: true, hasParameter: false, hasWildcard: false, minLength: 4,
			//	types: []rune{'.'}, positions: []int{1, 3}, sizes: []int{3}, static: []int{0}, parameter: nil}
			if tt.info != a.String() {
				t.Errorf("extractPathInfo(string) | invalid 'path'\n   actual: %v\n expected: %v", a.String(), tt.info)
			}
		})
	}
}

func Test_PathInfo_MaybeMatches(t *testing.T) {
	routes := []struct {
		first    string
		second   string
		expected bool
	}{
		{"/blog/category/page/subpage", "/blog/category/page/subpage", true},
		{"/blog/category/page/:subpage", "/blog/category/page/:subpage", true},
		{"/blog/category/page/*subpage", "/blog/category/page/*subpage", true},
		{"/blog/category/:page/:subpage", "/blog/category/:foo/:bar", true},
		{"/blog/category/:page/*subpage", "/blog/category/:foo/:bar", true},
		{"/blog/category/:page/*subpage", "/blog/category/:foo/:bar/x/y/z", true},
		{"/blog/:category/:page/*subpage", "/blog/category/page/:bar/x/y/z", true},
		{"/blog/:category/:page/:subpage", "/:blog/:category/*page", true},
		{"/blog/category/page/subpage", "/:blog/*category", true},
		{"/blog/category/page/subpage", "/blog/*category", true},
		{"/:blog/category/page/subpage", "/blog/*category", true},
		{"/:blog/:category/:page/:subpage", "/blog/*category", true},
		{"/blog/:category/:page/:subpage", "/:blog/:category/page", false},
		{"/blog/category-2/page/subpage", "/:blog/category-1/*page", false},
		{"/blog/category/page/subpage", "/blog/category/*page", true},
		{"/blog/category-1/page/subpage", "/blog/category-2/*page", false},
		{"/blog/category/page/subpage", "/blog/*category", true},
		{"/blog-1/category/page/subpage", "/blog-2/*category", false},
		{"/blog/*category", "/:blog/:category/*page", true},
		{"/blog/*category", "/blog/:category", true},
		{"/blog/*category", "/blog/", true},
		{"/blog/*category", "/blog", false},
		{"/blog", "/blog/*category", false},
		{"/:blog", "/blog/category/page/subpage", false},
		{"/*blog", "/blog/category/page/subpage", true},
		{"/", "", true},
	}
	for _, tt := range routes {
		t.Run(tt.first, func(t *testing.T) {
			first := ParseRouteInfo(tt.first)
			second := ParseRouteInfo(tt.second)
			matches := first.Matches(second)
			if matches != tt.expected {
				t.Errorf("RouteInfo.Matches(string) | invalid \n   actual: %v\n expected: %v", matches, tt.expected)
			}
		})
	}
}

func Test_PathInfo_Match(t *testing.T) {
	route := New()
	routes := []struct {
		route       string
		path        string
		match       bool
		paramNames  []string
		paramValues []string
	}{
		{"/blog/category/page/subpage", "/blog/category", false, nil, nil},
		{"/blog/category/page/subpage", "/blog/category/page/subpage", true, nil, nil},
		{"/blog/category/page/subpage", "/blog/category/page", false, nil, nil},
		{"/blog-1/category/page/subpage", "/blog-2/category/page/subpage", false, nil, nil},

		{"/blog/category-1/page/subpage", "/blog/category-2/page", false, nil, nil},
		{"/blog/category-1/page/subpage", "/blog/category-2/page/subpage", false, nil, nil},

		{"/blog/category/page/:subpage", "/blog/category/page/subpage", true, []string{"subpage"}, []string{"subpage"}},
		{"/blog/category/page/*subpage", "/blog/category/page/subpage", true, []string{"subpage"}, []string{"/subpage"}},

		{"/blog/category/:page/:subpage", "/blog/category/foo/bar", true, []string{"page", "subpage"}, []string{"foo", "bar"}},
		{"/blog/category/:page/*subpage", "/blog/category/foo/bar", true, []string{"page", "subpage"}, []string{"foo", "/bar"}},
		{"/blog/category/:page/*subpage", "/blog/category/foo/bar/x/y/z", true, []string{"page", "subpage"}, []string{"foo", "/bar/x/y/z"}},
		{"/blog/category-1/:page/:subpage", "/blog/category-2/page/subpage", false, nil, nil},

		{"/blog/:category/:page/*subpage", "/blog/category/page/bar/x/y/z", true, []string{"category", "page", "subpage"}, []string{"category", "page", "/bar/x/y/z"}},
		{"/blog/:category/:page/:subpage", "/blog/category/page", false, nil, nil},
		{"/blog/:category/:page/:subpage", "/blog/category/page", false, nil, nil},

		{"/:blog/category/page/subpage", "/blog/category", false, nil, nil},
		{"/:blog/:category/:page/:subpage", "/blog/category", false, nil, nil},

		{"/blog/*category", "/blog", false, nil, nil},
		{"/blog/*category", "/blog/", true, []string{"category"}, []string{"/"}},
		{"/blog/*category", "/blog/category", true, []string{"category"}, []string{"/category"}},
		{"/blog/*category", "/blog/category/page", true, []string{"category"}, []string{"/category/page"}},

		{"/blog/*", "/blog/", true, []string{"filepath"}, []string{"/"}},

		{"/blog", "/blog", true, nil, nil},
		{"/blog", "/blog-2", false, nil, nil},
		{"/blog", "/blog/category", false, nil, nil},
		{"/:blog", "/blog/category/page/subpage", false, nil, nil},
		{"/*blog", "/blog/category/page/subpage", true, []string{"blog"}, []string{"/blog/category/page/subpage"}},

		{"/", "/", true, nil, nil},
		{"/*", "/", true, []string{"filepath"}, []string{"/"}},
		{"/*", "/blog", true, []string{"filepath"}, []string{"/blog"}},
		{"/*path", "/blog", true, []string{"path"}, []string{"/blog"}},
		{"/*", "/blog/category", true, []string{"filepath"}, []string{"/blog/category"}},
		{"/*", "/blog/category/page/subpage", true, []string{"filepath"}, []string{"/blog/category/page/subpage"}},
	}
	for _, tt := range routes {
		t.Run(tt.route, func(t *testing.T) {
			info := ParseRouteInfo(tt.route)
			ctx := route.poolGetContext(nil, nil, tt.path)
			ctx.parsePathSegments()
			match, paramNames, paramValues := info.Match(ctx)
			if match != tt.match {
				t.Errorf("RouteInfo.Match() | invalid 'match'\n   actual: %v\n expected: %v", match, tt.match)
			} else {
				if !reflect.DeepEqual(paramNames, tt.paramNames) {
					t.Errorf("RouteInfo.Match() | invalid 'paramNames'\n   actual: %v\n expected: %v", paramNames, tt.paramNames)
				}
				if !reflect.DeepEqual(paramValues, tt.paramValues) {
					t.Errorf("RouteInfo.Match() | invalid 'paramValues'\n   actual: %v\n expected: %v", paramValues, tt.paramValues)
				}
			}
		})
	}
}
