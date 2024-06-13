// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/form.go
package chain

import (
	"errors"
	"net/http"
)

const defaultMemory = 32 << 20

type formBinding struct{}

func (formBinding) Bind(ctx *Context, obj any) error {
	req := ctx.Request
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := req.ParseMultipartForm(defaultMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return err
	}

	return mapFormByTag(obj, req.Form, "form")
}

type formPostBinding struct{}

func (formPostBinding) Bind(ctx *Context, obj any) error {
	req := ctx.Request
	if err := req.ParseForm(); err != nil {
		return err
	}

	return mapFormByTag(obj, req.PostForm, "form")
}

type formMultipartBinding struct{}

func (formMultipartBinding) Bind(ctx *Context, obj any) error {
	req := ctx.Request
	if err := req.ParseMultipartForm(defaultMemory); err != nil {
		return err
	}

	return mappingByPtr(obj, (*multipartRequest)(req), "form")
}
