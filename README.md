<br>
<div align="center">
    <img src="./docs/logo.png" />
    <p align="center">
        GO HTTP Router middleware
    </p>    
</div>

<a href="https://github.com/syntax-framework/syntax"><img width="160" src="./docs/logo-syntax.png" /></a>

**chain** is part of the [Syntax Framework](https://github.com/syntax-framework/syntax)

---

**chain** is a lightweight high performance HTTP request router (also called *multiplexer* or just *mux* for short)
for [Go](https://golang.org/).

In contrast to the [default mux](https://golang.org/pkg/net/http/#ServeMux) of Go's `net/http` package, this router
supports variables in the routing pattern and matches against the request method. It also scales better.

## Feature Overview

- Optimized HTTP router which smartly prioritize routes
- Build robust and scalable RESTful APIs
- Extensible middleware framework
- Handy functions to send variety of HTTP responses
- Centralized HTTP error handling

## Installation

```
go get github.com/syntax-framework/chain
```

## Example

```go
package main

import (
	"github.com/syntax-framework/chain"
	"log"
	"net/http"
)

func main() {
	router := chain.New()

	// Middleware
	router.Use(func(ctx *chain.Context, next func() error) error {
		println("first middleware")
		return next()
	})

	router.Use("GET", "/*", func(ctx *chain.Context) {
		println("second middleware")
	})

	// Handler
	router.GET("/", func(ctx *chain.Context) {
		ctx.Write([]byte("Hello World!"))
	})

	if err := http.ListenAndServe("localhost:8080", router); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}


```

[//]: # (- https://github.com/labstack/echo)
[//]: # (- https://github.com/go-playground/lars)
[//]: # (- https://github.com/gin-gonic/gin)
[//]: # (- https://github.com/aerogo/aero)
[//]: # (- https://github.com/gofiber/fiber)
