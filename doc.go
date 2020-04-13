// Copyright (c) 2020 Mikołaj Bednarek. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

/*
	Package smartapi allows to quickly implement solid REST APIs in Golang.
	The idea behind the project is to replace handler functions with ordinary looking functions.
	This allows service layer methods to be used as handlers.
	Designation of a dedicated API layer is still advisable in order to map errors to status codes, write cookies, headers, etc.

	SmartAPI is based on github.com/go-chi/chi. This allows Chi middlewares to be used.

	Examples

	This example returns a greeting with a name based on a query param `name`.

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

	It's possible to use even standard Go functions

	But it's a good practice to use your own handler functions for your api.

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

	You can use SmartAPI with service layer methods as shown here.

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


	Middlewares can be used just as in Chi. `Use(...)` appends middlewares to be used.
	`With(...)` creates a copy of a router with chosen middlewares.

	Routing works similarity to Chi routing. Parameters can be prepended to be used in all endpoints in that route.

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

	Support for legacy handlers

	Legacy handlers are supported with no overhead.
	They are directly passed as ordinary handler functions.
	No additional variadic arguments are required for legacy handler to be used.
*/
package smartapi
