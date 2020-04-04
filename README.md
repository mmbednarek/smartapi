# SmartAPI REST Library

[![GoDoc](https://godoc.org/github.com/mmbednarek/smartapi?status.svg)](https://godoc.org/github.com/mmbednarek/smartapi)
[![Build Status](https://travis-ci.org/mmbednarek/smartapi.svg?branch=master)](https://travis-ci.org/mmbednarek/smartapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmbednarek/smartapi)](https://goreportcard.com/report/github.com/mmbednarek/smartapi)
[![Coverage Status](https://coveralls.io/repos/github/mmbednarek/smartapi/badge.svg?branch=master&vs=1)](https://coveralls.io/github/mmbednarek/smartapi?branch=master)

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

	"github.com/mmbednarek/smartapi"
)

type API struct {
	*smartapi.Server
}

func (a *API) Init() {
	a.Get("/greeting", a.Greeting,
		smartapi.QueryParam("name"),
	)
}

func (a *API) Greeting(name string) (string, error) {
	return fmt.Sprintf("Hello %s!\n", name), nil
}

func main() {
	api := &API{
		Server: smartapi.NewServer(smartapi.DefaultLogger),
	}
	panic(smartapi.StartAPI(api, ":8080"))
}
```

```bash
$ curl '127.0.0.1:8080/greeting?name=Johnny'
Hello Johnny!
```

### Service example

You can use SmartAPI with service layer methods as shown here.

```go
package main

import (
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

func newAPI(service Service) *smartapi.Server {
	api := smartapi.NewServer(smartapi.DefaultLogger)

	api.Post("/user", service.RegisterUser,
		smartapi.JSONBody(User{}),
	)
	api.Post("/user/auth", service.Auth,
		smartapi.PostQueryParam("login"),
		smartapi.PostQueryParam("password"),
	)
	api.Get("/user", service.GetUserData,
		smartapi.Header("X-Session-ID"),
	)
	api.Patch("/user", service.UpdateUser,
		smartapi.Header("X-Session-ID"),
		smartapi.JSONBody(User{}),
	)

	return api
}

func main() {
	svr := service.NewService() // Your service implementation
	api := newAPI(svr)
	panic(api.Start(":8080"))
}
```

## Future features

Planned features that will be added really soon.

+ Routing
+ Struct argument with request data passed into fields by tags

## Support for legacy handlers

Legacy handlers are supported with no overhead.
They are directly passed as ordinary handler functions.
No additional variadic arguments are required for legacy handler to be used.

## Handler response

### Empty body response

A handler function with error only return argument will return empty response body with 204 NO CONTENT status as default.

```go
a.Post("/test", func() error {
    return nil
})
```

### String response

Returned string will we written directly into a function body.

```go
a.Get("/test", func() (string, error) {
    return "Hello World", nil
})
```

### Byte slice response

Just as with the string, the slice will we written directly into a function body.

```go
a.Get("/test", func() ([]byte, error) {
    return []byte("Hello World"), nil
})
```

### Struct, pointer or interface response

A struct, a pointer, an interface or a slice different than a byte slice with be encoded into a json format.

```go
a.Get("/test", func() (interface{}, error) {
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
a.Get("/order/{id}", func(orderID string) (*Order, error) {
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

### JSON Body

JSON Body unmarshals the request's body into a given structure type.
Expects a pointer to that structure as a function argument.

```go
a.Post("/user", func(u *User) error {
    return db.AddUser(u)
},
    smartapi.JSONBody(User{}),
)
```

### String Body

String body passes the request's body as a string.

```go
a.Post("/user", func(body string) error {
    fmt.Printf("Request body: %s\n", body)
    return nil
},
    smartapi.StringBody(),
)
```

### Byte Slice Body

Byte slice body passes the request's body as a byte slice.

```go
a.Post("/user", func(body []byte) error {
    fmt.Printf("Request body: %s\n", string(body))
    return nil
},
    smartapi.ByteSliceBody(),
)
```

### Body Reader

Byte reader body passes the io.Reader interface to read request's body.

```go
a.Post("/user", func(body io.Reader) error {
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
a.Post("/user", func(w http.ResponseWriter) error {
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
a.Post("/user", func(r *http.Request) error {
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
a.Get("/user", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.QueryParam("name"),
)
```

### URL param

URL uses read chi's URL param and passes it into a function as a string.

```go
a.Get("/user/{name}", func(name string) (*User, error) {
    return db.GetUser(name)
},
    smartapi.URLParam("name"),
)
```

### Header

Header reads the value of the selected request header and passes it as a string to function.

```go
a.Get("/example", func(test string) (string, error) {
    return fmt.Sprintf("The X-Test headers is %s", test), nil
},
    smartapi.Header("X-Test"),
)
```

### Required Header

Like header, but responds with 400 BAD REQUEST, if the header is not present.

```go
a.Get("/example", func(test string) (string, error) {
    return fmt.Sprintf("The X-Test headers is %s", test), nil
},
    smartapi.RequiredHeader("X-Test"),
)
```

### Cookie

Reads a cookie from the request and passes the value into a function as a string.

```go
a.Get("/example", func(c string) (string, error) {
    return fmt.Sprintf("cookie: %s", c)
},
    smartapi.Cookie("cookie"),
)
```

### Context

Context passes r.Context() into a function.

```go
a.Get("/example", func(ctx context.Context) (string, error) {
    return fmt.Sprintf("ctx: %s", ctx)
},
    smartapi.Context(),
)
```

### ResponseHeaders

Response headers allows an endpoint to add response headers.

```go
a.Get("/example", func(headers smartapi.Headers) error {
    headers.Set("Api-Version", "1.2.3")
    return nil
},
    smartapi.ResponseHeaders(),
)
```

### ResponseCookies

Response cookies allows an endpoint to easily add Set-Cookie header.

```go
a.Get("/example", func(cookies smartapi.Cookies) error {
    cookies.Add(&http.Cookie{Name: "Foo", Value: "Bar"})
    return nil
},
    smartapi.ResponseCookies(),
)
```

## Endpoint attributes

### Response status

ResponseStatus allows the response status to be set for endpoints with empty response body.
Default status is 204 NO CONTENT.

```go
a.Get("/example", func() error {
    return nil
},
    smartapi.ResponseStatus(http.StatusCreated),
)
```

### Middleware

Adds middlewares to the endpoint

```go
a.Get("/example", func() error {
    return nil
},
    smartapi.Middleware(middleware.DefaultLogger, middleware.SetHeader("Api-Version", "1.2.3")),
)
```

**NOTICE** You can add a middlewares to all the endpoints with `a.With(...)`.
