<br>
<div align="center">
    <img src="./docs/logo.png" />
    <p align="center">
        To create distributed systems in a simple, elegant and safe way.
    </p>    
</div>

<a href="https://github.com/syntax-framework/syntax"><img width="160" src="./docs/logo-syntax.png" /></a>

**chain** is part of the [Syntax Framework](https://github.com/syntax-framework/syntax)

---

Chain is a core library that seeks to provide all the necessary machinery to create distributed systems in a simple,
elegant and safe way.

## Feature Overview

- Optimized HTTP Router middleware
- Crypto-related functionality
    - **KeyGenerator**: PBKDF2 (Password-Based Key Derivation Function 2). It can be used to derive a number of keys for
      various purposes from a given secret. This lets applications have a single secure secret, but avoid reusing
      that key in multiple incompatible contexts.
    - **MessageVerifier**: makes it easy to generate and verify messages which are signed to prevent tampering.
    - **MessageEncryptor** is a simple way to encrypt values which get stored somewhere you don't trust.
- Realtime Publisher/Subscriber _ignore.service.
- Socket & Channels: A socket implementation that multiplexes messages over channels.

## Installation

```
go get github.com/syntax-framework/chain
```

## Router

**chain** has a lightweight high performance HTTP request router (also called *multiplexer* or just *mux* for short)
for [Go](https://golang.org/). In contrast to the [default mux](https://golang.org/pkg/net/http/#ServeMux) of
Go's `net/http` package, this router supports variables in the routing pattern and matches against the request method.
It also scales better.

- Optimized HTTP router which smartly prioritize routes
- Build robust and scalable RESTful APIs
- Extensible Middleware framework
- Handy functions to send variety of HTTP responses
- Centralized HTTP error handling

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

	// Grouping
	v1 := router.Group("/v1")
	{
		v1.GET("/users", func(ctx *chain.Context) {
			ctx.Write([]byte("[001]"))
		})
	}

	v2 := router.Group("/v2")
	{
		v2.GET("/users", func(ctx *chain.Context) {
			ctx.Write([]byte("[002]"))
		})
	}

	if err := http.ListenAndServe("localhost:8080", router); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
```

## Crypto

## PubSub

## Socket & Channels
