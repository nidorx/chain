// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// see: https://github.com/gin-gonic/gin/blob/master/binding/binding.go
package chain

import "net/http"

// Binding describes the interface which needs to be implemented for binding the
// data present in the request such as JSON request body, query parameters or
// the form POST.
type Binding interface {
	Bind(*Context, any) error
}

// These implement the Binding interface and can be used to bind the data
// present in the request to struct instances.
var (
	BindingXML           Binding = xmlBinding{}            // xml
	BindingJSON          Binding = jsonBinding{}           // json
	BindingPath          Binding = pathBinding{}           // path
	BindingForm          Binding = formBinding{}           // form
	BindingFormPost      Binding = formPostBinding{}       // form
	BindingFormMultipart Binding = formMultipartBinding{}  // form
	BindingQuery         Binding = queryBinding{}          // query
	BindingHeader        Binding = headerBinding{}         // header
	BindingDefault       Binding = &BindingDefaultStruct{} // query, json, xml, form
)

type BindingDefaultStruct struct {
	BindHeader bool
}

func (s *BindingDefaultStruct) Bind(ctx *Context, obj any) error {

	bb := []Binding{BindingQuery}

	if ctx.paramCount > 0 {
		bb = append(bb, BindingPath)
	}

	if s.BindHeader {
		bb = append(bb, BindingHeader)
	}

	if ctx.Request.Method != http.MethodGet {
		switch ctx.GetContentType() {
		case "application/json":
			bb = append(bb, BindingJSON)
		case "application/xml", "text/xml":
			bb = append(bb, BindingXML)
		case "multipart/form-data":
			bb = append(bb, BindingFormMultipart)
		default: // case "application/x-www-form-urlencoded":
			bb = append(bb, BindingForm)
		}
	}

	for _, b := range bb {
		if err := b.Bind(ctx, obj); err != nil {
			return err
		}
	}

	return nil
}

// Bind checks the Method and Content-Type to select a binding engine automatically,
// Depending on the "Content-Type" header different bindings are used, for example:
//
//	"application/json" --> JSON binding
//	"application/xml"  --> XML binding
//
// It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// It decodes the json payload into the struct specified as a pointer.
// It writes a 400 error and sets Content-Type header "text/plain" in the response if input is not valid.
func (ctx *Context) Bind(obj any) error {
	return ctx.MustBindWith(obj, BindingDefault)
}

// ShouldBind checks the Method and Content-Type to select a binding engine automatically,
// Depending on the "Content-Type" header different bindings are used, for example:
//
//	"application/json" --> JSON binding
//	"application/xml"  --> XML binding
//
// It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// It decodes the json payload into the struct specified as a pointer.
// Like c.Bind() but this method does not set the response status code to 400 or abort if input is not valid.
func (ctx *Context) ShouldBind(obj any) error {
	return ctx.ShouldBindWith(obj, BindingDefault)
}

// ShouldBindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (ctx *Context) ShouldBindWith(obj any, b Binding) error {
	if err := b.Bind(ctx, obj); err != nil {
		return err
	}

	return validate(obj)
}

// MustBindWith binds the passed struct pointer using the specified binding engine.
// It will abort the request with HTTP 400 if any error occurs.
// See the binding package.
func (ctx *Context) MustBindWith(obj any, b Binding) error {
	if err := ctx.ShouldBindWith(obj, b); err != nil {
		ctx.BadRequest()
		return err
	}
	return nil
}

// BindJSON is a shortcut for c.MustBindWith(obj, BindingJSON).
func (ctx *Context) BindJSON(obj any) error {
	return ctx.MustBindWith(obj, BindingJSON)
}

// ShouldBindJSON is a shortcut for c.ShouldBindWith(obj, BindingJSON).
func (c *Context) ShouldBindJSON(obj any) error {
	return c.ShouldBindWith(obj, BindingJSON)
}

// BindXML is a shortcut for c.MustBindWith(obj, binding.BindXML).
func (ctx *Context) BindXML(obj any) error {
	return ctx.MustBindWith(obj, BindingXML)
}

// ShouldBindXML is a shortcut for c.ShouldBindWith(obj, BindingXML).
func (c *Context) ShouldBindXML(obj any) error {
	return c.ShouldBindWith(obj, BindingXML)
}

// BindPath is a shortcut for c.MustBindWith(obj, BindingPath).
func (ctx *Context) BindPath(obj any) error {
	return ctx.MustBindWith(obj, BindingPath)
}

// ShouldBindPath is a shortcut for c.ShouldBindWith(obj, BindingPath).
func (ctx *Context) ShouldBindPath(obj any) error {
	return ctx.ShouldBindWith(obj, BindingPath)
}

// BindQuery is a shortcut for c.MustBindWith(obj, BindingQuery).
func (ctx *Context) BindQuery(obj any) error {
	return ctx.MustBindWith(obj, BindingQuery)
}

// ShouldBindQuery is a shortcut for c.ShouldBindWith(obj, BindingQuery).
func (ctx *Context) ShouldBindQuery(obj any) error {
	return ctx.ShouldBindWith(obj, BindingQuery)
}

// BindHeader is a shortcut for c.MustBindWith(obj, BindingHeader).
func (ctx *Context) BindHeader(obj any) error {
	return ctx.MustBindWith(obj, BindingHeader)
}

// ShouldBindHeader is a shortcut for c.ShouldBindWith(obj, BindingHeader).
func (ctx *Context) ShouldBindHeader(obj any) error {
	return ctx.ShouldBindWith(obj, BindingHeader)
}

// BindForm is a shortcut for c.MustBindWith(obj, BindingForm).
func (ctx *Context) BindForm(obj any) error {
	return ctx.MustBindWith(obj, BindingForm)
}

// ShouldBindForm is a shortcut for c.ShouldBindWith(obj, BindingForm).
func (ctx *Context) ShouldBindForm(obj any) error {
	return ctx.ShouldBindWith(obj, BindingForm)
}

// BindFormPost is a shortcut for c.MustBindWith(obj, BindingFormPost).
func (ctx *Context) BindFormPost(obj any) error {
	return ctx.MustBindWith(obj, BindingFormPost)
}

// ShouldBindFormPost is a shortcut for c.ShouldBindWith(obj, BindingFormPost).
func (ctx *Context) ShouldBindFormPost(obj any) error {
	return ctx.ShouldBindWith(obj, BindingFormPost)
}

// BindFormMultipart is a shortcut for c.MustBindWith(obj, BindingFormMultipart).
func (ctx *Context) BindFormMultipart(obj any) error {
	return ctx.MustBindWith(obj, BindingFormMultipart)
}

// ShouldBindFormMultipart is a shortcut for c.ShouldBindWith(obj, BindingFormMultipart).
func (ctx *Context) ShouldBindFormMultipart(obj any) error {
	return ctx.ShouldBindWith(obj, BindingFormMultipart)
}
