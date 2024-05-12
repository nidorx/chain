// Copyright 2018 Gin Core Team. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/uri.go
package chain

type pathBinding struct{}

func (pathBinding) Bind(ctx *Context, obj any) error {
	m := make(map[string][]string, ctx.paramCount)
	for i := 0; i < ctx.paramCount; i++ {
		m[ctx.paramNames[i]] = []string{ctx.paramValues[i]}
	}

	return mapFormByTag(obj, m, "path")
}
