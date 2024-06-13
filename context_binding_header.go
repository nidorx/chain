// Copyright 2022 Gin Core Team. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/header.go
package chain

import (
	"net/textproto"
	"reflect"
)

type headerBinding struct{}

func (headerBinding) Bind(ctx *Context, obj any) error {
	return mapHeader(obj, ctx.Request.Header)
}

func mapHeader(ptr any, h map[string][]string) error {
	return mappingByPtr(ptr, headerSource(h), "header")
}

type headerSource map[string][]string

var _ setter = headerSource(nil)

func (hs headerSource) TrySet(value reflect.Value, field reflect.StructField, tagValue string, opt setOptions) (bool, error) {
	return setByForm(value, field, hs, textproto.CanonicalMIMEHeaderKey(tagValue), opt)
}
