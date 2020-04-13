# SmartAPI REST Library

[![GoDoc](https://godoc.org/github.com/mmbednarek/smartapi?status.svg)](https://godoc.org/github.com/mmbednarek/smartapi)
[![Build Status](https://travis-ci.org/mmbednarek/smartapi.svg?branch=master)](https://travis-ci.org/mmbednarek/smartapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmbednarek/smartapi)](https://goreportcard.com/report/github.com/mmbednarek/smartapi)
[![Coverage Status](https://coveralls.io/repos/github/mmbednarek/smartapi/badge.svg?branch=master&smartapi=true)](https://coveralls.io/github/mmbednarek/smartapi?branch=master)

SmartAPI allows to quickly implement solid REST APIs in Golang.
The idea behind the project is to replace handler functions with ordinary looking functions.
This allows service layer methods to be used as handlers.
Designation of a dedicated API layer is still advisable in order to map errors to status codes, write cookies, headers, etc.

SmartAPI is based on [github.com/go-chi/chi](https://github.com/go-chi/chi). This allows Chi middlewares to be used.

## Examples

This example returns a greeting with a name based on a query param `name`.

```go
package main

import (
    "fmt" 
    "log"
    "net/http"

    "github.com/mmbednarek/smartapi"
)

func MountAPI() http.Handler {
    r := smartapi.NewRouter()
	r.Get("/greeting", Greeting,
		smartapi.QueryParam("name"),
	)

    return r.MustHandler()
}

func Greeting(name string) string {
	return fmt.Sprintf("Hello %s!\n", name)
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", MountAPI()))
}
```

```bash
$ curl '127.0.0.1:8080/greeting?name=Johnny'
Hello Johnny!
```

## It's possible to use even standard Go functions

But it's a good practice to use your own handler functions for your api.

```go
package main

import (
    "encoding/base32"
    "encoding/base64"
    "log"
    "net/http"

    "github.com/mmbednarek/smartapi"
)

func MountAPI() http.Handler {
    r := smartapi.NewRouter()

    r.Route("/encode", func(r smartapi.Router) {
        r.Post("/base64", base64.StdEncoding.EncodeToString)
        r.Post("/base32", base32.StdEncoding.EncodeToString)
    }, smartapi.ByteSliceBody())
    r.Route("/decode", func(r smartapi.Router) {
        r.Post("/base64", base64.StdEncoding.DecodeString)
        r.Post("/base32", base32.StdEncoding.DecodeString)
    }, smartapi.StringBody())

    return r.MustHandler()
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", MountAPI()))
}
```

```bash
~ $ curl 127.0.0.1:8080/encode/base64 -d 'smartAPI'
c21hcnRBUEk=
~ $ curl 127.0.0.1:8080/encode/base32 -d 'smartAPI'
ONWWC4TUIFIES===
~ $ curl 127.0.0.1:8080/decode/base64 -d 'c21hcnRBUEk='
smartAPI
~ $ curl 127.0.0.1:8080/decode/base32 -d 'ONWWC4TUIFIES==='
smartAPI
```

### Service example

You can use SmartAPI with service layer methods as shown here.

```go
package main

import (
    "log"
    "net/http"


    "github.com/mmbednarek/smartapi"
)

type Date struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

type User struct {
	Login       string `json:"login"`
	Password    string `json:"password,omitempty"`
	Email       string `json:"email"`
	DateOfBirth Date   `json:"date_of_birth"`
}

type Service interface {
	RegisterUser(user *User) error
	Auth(login, password string) (string, error)
	GetUserData(session string) (*User, error)
	UpdateUser(session string, user *User) error
}

func newHandler(service Service) http.Handler {
	r := smartapi.NewRouter()

	r.Post("/user", service.RegisterUser,
		smartapi.JSONBody(User{}),
	)
	r.Post("/user/auth", service.Auth,
		smartapi.PostQueryParam("login"),
		smartapi.PostQueryParam("password"),
	)
	r.Get("/user", service.GetUserData,
		smartapi.Header("X-Session-ID"),
	)
	r.Patch("/user", service.UpdateUser,
		smartapi.Header("X-Session-ID"),
		smartapi.JSONBody(User{}),
	)

	return r.MustHandler()
}

func main() {
	svr := service.NewService() // Your service implementation
	log.Fatal(http.ListenAndServe(":8080", newHandler(svr)))
}
```

## Middlewares

Middlewares can be used just as in Chi. `Use(...)` appends middlewares to be used.
`With(...)` creates a copy of a router with chosen middlewares.

## Routing

Routing works similarity to Chi routing. Parameters can be prepended to be used in all endpoints in that route.

```go
r.Route("/v1/foo", func(r smartapi.Router) {
    r.Route("/bar", func(r smartapi.Router) {
        r.Get("/test", func(ctx context.Context, foo string, test string) {
            ...
        },
            smartapi.QueryParam("test"),
        )
    },
        smartapi.Header("X-Foo"),
    )
},
    smartapi.Context(),
)
```

## Support for legacy handlers

Legacy handlers are supported with no overhead.
They are directly passed as ordinary handler functions.
No additional variadic arguments are required for legacy handler to be used.

## Handler response

### Empty body response

A handler function with error only return argument will return empty response body with 204 NO CONTENT status as default.

```go
r.Post("/test", func() error {
    return nil
})
```

### String response

Returned string will we written directly into a function body.

```go
r.Get("/test", func() (string, error) {
    return "Hello World", nil
})
```

### Byte slice response

Just as with the string, the slice will we written directly into a function body.

```go
r.Get("/test", func() ([]byte, error) {
    return []byte("Hello World"), nil
})
```

### Struct, pointer or interface response

A struct, a pointer, an interface or a slice different than a byte slice with be encoded into a json format.

```go
r.Get("/test", func() (interface{}, error) {
    return struct{
        Name string `json:"name"`
        Age  int    `json:"age"`
    }{"John", 34}, nil
})
```

## Errors

To return an error with a status code you can use one of the error functions: `smartapi.Error(status int, msg, reason string)`, `smartapi.Errorf(status int, msg string, fmt ...interface{})`, `smartapi.WrapError(status int, err error, reason string)`.
The API error contains an error message and an error reason. The message will be printed with a logger.
The reason will be returned in the response body. You can also return ordinary errors. They are treated as if their status code was 500.

```go
r.Get("/order/{id}", func(orderID string) (*Order, error) {
    order, err := db.GetOrder(orderID)
    if err != nil {
        if errors.Is(err, ErrNoSuchOrder) {
            return nil, smartapi.WrapError(http.StatusNotFound, err, "no such order")
        }
        return nil, err
    }
    return order, nil
},
    smartapi.URLParam("id"),
)
```

```bash
$ curl -i 127.0.0.1:8080/order/someorder
HTTP/1.1 404 Not Found
Date: Sun, 22 Mar 2020 14:17:34 GMT
Content-Length: 40
Content-Type: text/plain; charset=utf-8

{"status":404,"reason":"no such order"}
```

## Endpoint arguments

List of available endpoint attributes

### Request Struct

Request can be passed into a structure's field by tags.

```go
type headers struct {
    Foo string `smartapi:"header=X-Foo"`
    Bar string `smartapi:"header=X-Bar"`
}
r.Post("/user", func(h *headers) (string, error) {
    return fmt.Sprintf("Foo: %s, Bar: %s\n", h.Foo, h.Bar), nil
},
    smartapi.RequestStruct(headers{}),
)
```

Every argument has a tag value equivalent

| Tag Value   | Function Equivalent  | Expected Type |
|-------------|----------------------|---------------|
| `header=name` | `Header("name")`  | `string` |
| `r_header=name` | `RequiredHeader("name")`  | `string` |
| `json_body`   | `JSONBody()`  | `...` |
| `string_body`   | `StringBody()`  | `string` |
| `byte_slice_body`   | `ByteSliceBody()`  | `[]byte` |
| `body_reader`   | `BodyReader()`  | `io.Reader` |
| `url_param=name` | `URLParam("name")`  | `string` |
| `context`   | `Context()`  | `context.Context` |
| `query_param=name`   | `QueryParam("name")`  | `string` |
| `r_query_param=name`   | `RequiredQueryParam("name")`  | `string` |
| `post_query_param=name`   | `PostQueryParam("name")`  | `string` |
| `r_post_query_param=name`   | `RequiredPostQueryParam("name")`  | `string` |
| `cookie=name`   | `Cookie("name")`  | `string` |
| `r_cookie=name`   | `RequiredCookie("name")`  | `string` |
| `response_headers`   | `ResponseHeaders()`  | `smartapi.Headers` |
| `response_cookies`   | `ResponseCookies()`  | `smartapi.Cookies` |
| `response_writer`   | `ResponseWriter()`  | `http.ResponseWriter` |
| `request`   | `Request()`  | `*http.Request` |
| `request_struct`   | `RequestStruct()`  | `struct{...}` |
| `as_int=header=name`   | `AsInt(Header("name")`  | `int` |
| `as_byte_slice=header=name`   | `AsByteSlice(Header("name")`  | `[]byte` |

### JSON Body

JSON Body unmarshals the request's body into a given structure type.
Expects a pointer to that structure as a function argument.
If you want to use the object directly (not as a pointer) you can use JSONBodyDirect.

```go
r.Post("/user", func(u *User) error {
    return db.AddUser(u)
},
    smartapi.JSONBody(User{}),
)
```

### String Body

String body passes the request's body as a string.

```go
r.Post("/user", func(body string) error {
    fmt.Printf("Request body: %s\n", body)
    return nil
},
    smartapi.StringBody(),
)
```

### Byte Slice Body

Byte slice body passes the request's body as a byte slice.

```go
r.Post("/user", func(body []byte) error {
    fmt.Printf("Request body: %s\n", string(body))
    return nil
},
    smartapi.ByteSliceBody(),
)
```

### Body Reader

Byte reader body passes the io.Reader interface to read request's body.

```go
r.Post("/user", func(body io.Reader) error {
    buff, err := ioutil.ReadAll()
    if err != nil {
        return err
    }
    return nil
},
    smartapi.BodyReader(),
)
```

### Response Writer

Classic `http.ResponseWriter` can be used as well.

```go
r.Post("/user", func(w http.ResponseWriter) error {
    _, err := w.Write([]byte("RESPONSE"))
    if err != nil {
        return err
    }
    return nil
},
    smartapi.ResponseWriter(),
)
```

### Request

Classic `*http.Request` can be passed as an argument.

```go
r.Post("/user", func(r *http.Request) error {
    buff, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return err
    }
    fmt.Printf("Request body is: %s\n", string(buff))
    return nil
},
    smartapi.Request(),
)
```

### Query param

Query param reads the value of the selected param and passes it as a string to function.

```go
r.Get("/user", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.QueryParam("name"),
)
```

### Required Query param

Like `QueryParam()` but returns 400 BAD REQUEST when empty.

```go
r.Get("/user", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.RequiredQueryParam("name"),
)
```

### Post Query param

Reads a query param from requests body.

```go
r.Get("/user", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.PostQueryParam("name"),
)
```


### Required Post Query param

Like `PostQueryParam()` but returns 400 BAD REQUEST when empty.

```go
r.Get("/user", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.RequiredPostQueryParam("name"),
)
```


### URL param

URL uses read chi's URL param and passes it into a function as a string.

```go
r.Get("/user/{name}", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.URLParam("name"),
)
```

### Header

Header reads the value of the selected request header and passes it as a string to function.

```go
r.Get("/example", func(test string) (string, error) {
    return fmt.Sprintf("The X-Test headers is %s", test), nil
},
    smartapi.Header("X-Test"),
)
```

### Required Header

Like `Header()`, but responds with 400 BAD REQUEST, if the header is not present.

```go
r.Get("/example", func(test string) (string, error) {
    return fmt.Sprintf("The X-Test headers is %s", test), nil
},
    smartapi.RequiredHeader("X-Test"),
)
```

### Cookie

Reads a cookie from the request and passes the value into a function as a string.

```go
r.Get("/example", func(c string) (string, error) {
    return fmt.Sprintf("cookie: %s", c)
},
    smartapi.Cookie("cookie"),
)
```

### Required Cookie

Like `Cookie()`, but returns 400 BAD REQUEST when empty.

```go
r.Get("/example", func(c string) (string, error) {
    return fmt.Sprintf("cookie: %s", c)
},
    smartapi.RequiredCookie("cookie"),
)
```

### Context

Context passes r.Context() into a function.

```go
r.Get("/example", func(ctx context.Context) (string, error) {
    return fmt.Sprintf("ctx: %s", ctx)
},
    smartapi.Context(),
)
```

### ResponseHeaders

Response headers allows an endpoint to add response headers.

```go
r.Get("/example", func(headers smartapi.Headers) error {
    headers.Set("Api-Version", "1.2.3")
    return nil
},
    smartapi.ResponseHeaders(),
)
```

### ResponseCookies

Response cookies allows an endpoint to easily add Set-Cookie header.

```go
r.Get("/example", func(cookies smartapi.Cookies) error {
    cookies.Add(&http.Cookie{Name: "Foo", Value: "Bar"})
    return nil
},
    smartapi.ResponseCookies(),
)
```

## Casts

Request attributes can be automatically casted to desired type.

### AsInt

```go
r.Get("/example", func(value int) error {
    ...
    return nil
},
    smartapi.AsInt(smartapi.Header("Value")),
)
```

### AsByteSlice

```go
r.Get("/example", func(value []byte) error {
    ...
    return nil
},
    smartapi.AsByteSlice(smartapi.Header("Value")),
)
```

Conversion to int

## Endpoint attributes

### Response status

ResponseStatus allows the response status to be set for endpoints with empty response body.
Default status is 204 NO CONTENT.

```go
r.Get("/example", func() error {
    return nil
},
    smartapi.ResponseStatus(http.StatusCreated),
)
```
