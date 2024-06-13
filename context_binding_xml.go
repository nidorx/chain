// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/xml.go
package chain

import (
	"bytes"
	"encoding/xml"
)

type xmlBinding struct{}

func (xmlBinding) Bind(ctx *Context, obj any) (err error) {
	var body []byte
	if body, err = ctx.BodyBytes(); err != nil {
		return err
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	return decoder.Decode(obj)
}
