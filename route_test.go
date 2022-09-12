package chain

import (
	"reflect"
	"testing"
)

func Test_PathInfo_extract(t *testing.T) {
	routes := []struct {
		path string
		info PathDetails
	}{
		{path: "/", info: PathDetails{
			path: "/", hasStatic: true, hasParameter: false, hasWildcard: false, params: nil, priority: 3,
			segments: []string{""}},
		},
		{path: "/doc/", info: PathDetails{
			path: "/doc/", hasStatic: true, hasParameter: false, hasWildcard: false, priority: 15, params: nil,
			segments: []string{"doc", ""}},
		},
		{path: "/search/", info: PathDetails{
			path: "/search/", hasStatic: true, hasParameter: false, hasWildcard: false, priority: 15, params: nil,
			segments: []string{"search", ""}},
		},
		{path: "/user/:name", info: PathDetails{
			path: "/user/:name", hasStatic: true, hasParameter: true, hasWildcard: false, priority: 14,
			params: []string{"name"}, segments: []string{"user", ":"}},
		},
		{path: "/search/:query", info: PathDetails{
			path: "/search/:query", hasStatic: true, hasParameter: true, hasWildcard: false, priority: 14,
			params: []string{"query"}, segments: []string{"search", ":"}},
		},
		{path: "/cmd/:tool/", info: PathDetails{
			path: "/cmd/:tool/", hasStatic: true, hasParameter: true, hasWildcard: false, priority: 38,
			params: []string{"tool"}, segments: []string{"cmd", ":", ""}},
		},
		{path: "/src/*filepath", info: PathDetails{
			path: "/src/*filepath", hasStatic: true, hasParameter: false, hasWildcard: true, priority: 13,
			params: []string{"filepath"}, segments: []string{"src", "*"}},
		},
		{path: "/user/:name/about", info: PathDetails{
			path: "/user/:name/about", hasStatic: true, hasParameter: true, hasWildcard: false, priority: 38,
			params: []string{"name"}, segments: []string{"user", ":", "about"}},
		},
		//"/info/:user/public",
		//"/cmd/:tool/:sub"
		//"/files/:dir/*filepath",
		{path: "/files/:dir/*filepath", info: PathDetails{
			path: "/files/:dir/*filepath", hasStatic: true, hasParameter: true, hasWildcard: true, priority: 36,
			params: []string{"dir", "filepath"}, segments: []string{"files", ":", "*"}},
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
			e := &tt.info
			a := extractPathDetails(tt.path)
			//PathDetails{
			//	path: "/doc/", segments: 1, hasStatic: true, hasParameter: false, hasWildcard: false, minLength: 4,
			//	types: []rune{'.'}, positions: []int{1, 3}, sizes: []int{3}, static: []int{0}, parameter: nil}
			if e.path != a.path {
				t.Errorf("extractPathInfo(string) | invalid 'path'\n   actual: %v\n expected: %v", a.path, e.path)
			}
			if !reflect.DeepEqual(e.segments, a.segments) {
				t.Errorf("extractPathInfo(string) | invalid 'segments'\n   actual: %v\n expected: %v", a.segments, e.segments)
			}
			if e.hasStatic != a.hasStatic {
				t.Errorf("extractPathInfo(string) | invalid 'hasStatic'\n   actual: %v\n expected: %v", a.hasStatic, e.hasStatic)
			}
			if e.hasParameter != a.hasParameter {
				t.Errorf("extractPathInfo(string) | invalid 'hasParameter'\n   actual: %v\n expected: %v", a.hasParameter, e.hasParameter)
			}
			if e.hasWildcard != a.hasWildcard {
				t.Errorf("extractPathInfo(string) | invalid 'hasWildcard'\n   actual: %v\n expected: %v", a.hasWildcard, e.hasWildcard)
			}
			if e.priority != a.priority {
				t.Errorf("extractPathInfo(string) | invalid 'priority'\n   actual: %v\n expected: %v", a.priority, e.priority)
			}
			if !reflect.DeepEqual(e.params, a.params) {
				t.Errorf("extractPathInfo(string) | invalid 'params'\n   actual: %v\n expected: %v", a.params, e.params)
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
			first := extractPathDetails(tt.first)
			second := extractPathDetails(tt.second)
			matches := first.MaybeMatches(second)
			if matches != tt.expected {
				t.Errorf("PathDetails.Matches(string) | invalid \n   actual: %v\n expected: %v", matches, tt.expected)
			}
		})
	}
}

func Test_PathInfo_Match(t *testing.T) {
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
			info := extractPathDetails(tt.route)
			ctx := &Context{path: tt.path}
			ctx.parsePathSegments()
			match, paramNames, paramValues := info.Match(ctx)
			if match != tt.match {
				t.Errorf("PathDetails.Match() | invalid 'match'\n   actual: %v\n expected: %v", match, tt.match)
			} else {
				if !reflect.DeepEqual(paramNames, tt.paramNames) {
					t.Errorf("PathDetails.Match() | invalid 'paramNames'\n   actual: %v\n expected: %v", paramNames, tt.paramNames)
				}
				if !reflect.DeepEqual(paramValues, tt.paramValues) {
					t.Errorf("PathDetails.Match() | invalid 'paramValues'\n   actual: %v\n expected: %v", paramValues, tt.paramValues)
				}
			}
		})
	}
}
