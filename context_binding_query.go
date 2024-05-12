// Copyright 2017 Manu Martinez-Almeida. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/query.go
package chain

type queryBinding struct{}

func (queryBinding) Bind(ctx *Context, obj any) error {
	values := ctx.Request.URL.Query()
	return mapFormByTag(obj, values, "query")
}
