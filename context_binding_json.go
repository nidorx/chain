// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/json.go
package chain

import (
	"bytes"
	"encoding/json"
)

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// any as a Number instead of as a float64.
var EnableDecoderUseNumber = false

// EnableDecoderDisallowUnknownFields is used to call the DisallowUnknownFields method
// on the JSON Decoder instance. DisallowUnknownFields causes the Decoder to
// return an error when the destination is a struct and the input contains object
// keys which do not match any non-ignored, exported fields in the destination.
var EnableDecoderDisallowUnknownFields = false

type jsonBinding struct{}

func (jsonBinding) Bind(ctx *Context, obj any) (err error) {
	var body []byte
	if body, err = ctx.BodyBytes(); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	return decoder.Decode(obj)
}
