package smartapi

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAttributes(t *testing.T) {
	type test struct {
		name         string
		request      func() *http.Request
		api          func(api *Server)
		responseCode int
		responseBody []byte
		checkHeader  func(h http.Header)
	}

	tests := []test{
		{
			name: "JSONBody",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name": "John", "surname": "Smith"}`)))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				type foo struct {
					Name    string `json:"name"`
					Surname string `json:"surname"`
				}
				api.Post("/test", func(f *foo) error {
					require.Equal(t, f.Name, "John")
					require.Equal(t, f.Surname, "Smith")
					return nil
				},
					JSONBody(foo{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Header",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("X-Test1", "value")
				request.Header.Set("X-Test2", "eulav")
				return request
			},
			api: func(api *Server) {
				api.Post("/test", func(test1, test2 string) error {
					require.Equal(t, test1, "value")
					require.Equal(t, test2, "eulav")
					return nil
				},
					Header("X-Test1"),
					Header("X-Test2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Query Params",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test?param2=value&param1=eulav", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func(param1, param2 string) error {
					require.Equal(t, param1, "eulav")
					require.Equal(t, param2, "value")
					return nil
				},
					QueryParam("param1"),
					QueryParam("param2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Query Params Error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test?a=%Z", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func(param1 string) error {
					return nil
				},
					QueryParam("a"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"could not parse form"}` + "\n"),
		},
		{
			name: "URL Params",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test/foo/orders/bar", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test/{param1}/orders/{param2}", func(param1, param2 string) error {
					require.Equal(t, param1, "foo")
					require.Equal(t, param2, "bar")
					return nil
				},
					URLParam("param1"),
					URLParam("param2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Cookies",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}

				request.AddCookie(&http.Cookie{
					Name:  "Test1",
					Value: "foo",
				})
				request.AddCookie(&http.Cookie{
					Name:  "Test2",
					Value: "bar",
				})

				return request
			},
			api: func(api *Server) {
				api.Get("/test", func(c1, c2 string) error {
					require.Equal(t, c1, "foo")
					require.Equal(t, c2, "bar")
					return nil
				},
					Cookie("Test1"),
					Cookie("Test2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Missing cookies",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func(cookie string) error {
					return nil
				},
					Cookie("Test1"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing cookie Test1"}` + "\n"),
		},
		{
			name: "Write header",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}

				return request
			},
			api: func(api *Server) {
				api.Get("/test", func(headers Headers) error {
					headers.Set("Test1", "foo")
					headers.Set("Test2", "bar")
					return nil
				},
					ResponseHeaders(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
			checkHeader: func(h http.Header) {
				require.Equal(t, h.Get("Test1"), "foo")
				require.Equal(t, h.Get("Test2"), "bar")
			},
		},
		{
			name: "Write cookies",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}

				return request
			},
			api: func(api *Server) {
				api.Get("/test", func(cookies Cookies) error {
					cookies.Add(&http.Cookie{
						Name:    "Test1",
						Value:   "foo",
						Expires: time.Unix(1584905348, 0),
					})
					cookies.Add(&http.Cookie{
						Name:    "Test2",
						Value:   "bar",
						Expires: time.Unix(1584905379, 0),
					})
					return nil
				},
					ResponseCookies(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
			checkHeader: func(h http.Header) {
				require.Equal(t, h.Get("Set-Cookie"), "Test1=foo; Expires=Sun, 22 Mar 2020 19:29:08 GMT; Test2=bar; Expires=Sun, 22 Mar 2020 19:29:39 GMT")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request()
			api := NewServer(nil)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, r.Code, tt.responseCode)
			require.Equal(t, r.Body, bytes.NewBuffer(tt.responseBody))

			if tt.checkHeader != nil {
				tt.checkHeader(r.Header())
			}
		})
	}
}

func TestHandlers(t *testing.T) {
	type test struct {
		name         string
		request      func() *http.Request
		api          func(api *Server)
		responseCode int
		responseBody []byte
	}

	tests := []test{
		{
			name: "Error Only Handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Error Only Handler Response Accepted",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() error {
					return nil
				},
					ResponseStatus(http.StatusAccepted),
				)
			},
			responseCode: http.StatusAccepted,
			responseBody: nil,
		},
		{
			name: "String Handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() (string, error) {
					return "foobar", nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte("foobar"),
		},
		{

			name: "Byte Slice Handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() ([]byte, error) {
					return []byte{1, 2, 45, 23}, nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte{1, 2, 45, 23},
		},
		{
			name: "Struct handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				type bar struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				type foo struct {
					Field1 string `json:"field1"`
					Field2 bar    `json:"field2"`
				}

				api.Get("/test", func() (foo, error) {
					return foo{
						Field1: "foo",
						Field2: bar{
							Field1: "bar",
							Field2: "foo",
						},
					}, nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte(`{"field1":"foo","field2":{"field1":"bar","field2":"foo"}}` + "\n"),
		},
		{
			name: "Pointer handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				type bar struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				type foo struct {
					Field1 string `json:"field1"`
					Field2 bar    `json:"field2"`
				}

				api.Get("/test", func() (*foo, error) {
					return &foo{
						Field1: "foo",
						Field2: bar{
							Field1: "bar",
							Field2: "foo",
						},
					}, nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte(`{"field1":"foo","field2":{"field1":"bar","field2":"foo"}}` + "\n"),
		},
		{
			name: "Interface handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				type bar struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				type foo struct {
					Field1 string `json:"field1"`
					Field2 bar    `json:"field2"`
				}

				api.Get("/test", func() (interface{}, error) {
					return &foo{
						Field1: "foo",
						Field2: bar{
							Field1: "bar",
							Field2: "foo",
						},
					}, nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte(`{"field1":"foo","field2":{"field1":"bar","field2":"foo"}}` + "\n"),
		},
		{
			name: "Slice handler",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() ([]string, error) {
					return []string{
						"foo",
						"bar",
						"rab",
						"oof",
					}, nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte(`["foo","bar","rab","oof"]` + "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request()
			api := NewServer(nil)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, r.Code, tt.responseCode)
			require.Equal(t, r.Body, bytes.NewBuffer(tt.responseBody))
		})
	}
}

func TestHandlersErrors(t *testing.T) {
	type test struct {
		name   string
		api    func(api *Server)
		expect error
	}

	tests := []test{
		{
			name: "Too many arguments",
			api: func(api *Server) {
				api.Get("/test", func(value string) error {
					return nil
				})
			},
			expect: errors.New("endpoint /test: number of arguments of a function doesn't match provided arguments"),
		},
		{
			name: "Too little arguments",
			api: func(api *Server) {
				api.Get("/test", func() error {
					return nil
				},
					QueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: number of arguments of a function doesn't match provided arguments"),
		},
		{
			name: "Non function handler",
			api: func(api *Server) {
				api.Get("/test", 456)
			},
			expect: errors.New("endpoint /test: handler must be a function"),
		},
		{
			name: "Invalid return type",
			api: func(api *Server) {
				api.Get("/test", func() int {
					return 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 2",
			api: func(api *Server) {
				api.Get("/test", func() (string, int) {
					return "", 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 3",
			api: func(api *Server) {
				api.Get("/test", func() (struct{}, int) {
					return struct{}{}, 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 4",
			api: func(api *Server) {
				api.Get("/test", func() (*struct{}, int) {
					return &struct{}{}, 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 5",
			api: func(api *Server) {
				api.Get("/test", func() ([]byte, int) {
					return []byte(""), 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "QueryParam wrong type",
			api: func(api *Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					QueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: expected a string type"),
		},
		{
			name: "URLParam wrong type",
			api: func(api *Server) {
				api.Get("/test/{name}", func(value int) error {
					return nil
				},
					URLParam("name"),
				)
			},
			expect: errors.New("endpoint /test/{name}: expected a string type"),
		},
		{
			name: "Header wrong type",
			api: func(api *Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					Header("name"),
				)
			},
			expect: errors.New("endpoint /test: expected a string type"),
		},
		{
			name: "Cookie wrong type",
			api: func(api *Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					Cookie("name"),
				)
			},
			expect: errors.New("endpoint /test: expected a string type"),
		},
		{
			name: "Invalid return value type",
			api: func(api *Server) {
				api.Get("/test", func() (func(), error) {
					return func() {
					}, nil
				})
			},
			expect: errors.New("endpoint /test: unsupported return type"),
		},
		{
			name: "Too many return arguments",
			api: func(api *Server) {
				api.Get("/test", func() (string, string, error) {
					return "", "", nil
				})
			},
			expect: errors.New("endpoint /test: invalid number of return arguments"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := NewServer(nil)
			tt.api(api)
			_, err := api.Handler()
			require.Equal(t, err, tt.expect)
		})
	}
}

func TestError(t *testing.T) {
	type test struct {
		name         string
		request      func() *http.Request
		api          func(api *Server)
		responseCode int
		responseBody []byte
	}

	tests := []test{
		{
			name: "Error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() error {
					return Error(http.StatusForbidden, "message", "reason")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"reason"}` + "\n"),
		},
		{
			name: "Errorf",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() error {
					return Errorf(http.StatusForbidden, "message: %s!", "format")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"unknown"}` + "\n"),
		},
		{
			name: "WrapError",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *Server) {
				api.Get("/test", func() error {
					return WrapError(http.StatusForbidden, errors.New("error"), "reason")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"reason"}` + "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request()
			api := NewServer(nil)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, r.Code, tt.responseCode)
			require.Equal(t, r.Body, bytes.NewBuffer(tt.responseBody))
		})
	}
}
