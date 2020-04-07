package smartapi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mmbednarek/smartapi"
	"github.com/mmbednarek/smartapi/mocks"
	"github.com/stretchr/testify/require"
)

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("just errors")
}

func TestAttributes(t *testing.T) {
	type test struct {
		name         string
		request      func() *http.Request
		api          func(api *smartapi.Server)
		responseCode int
		responseBody []byte
		checkHeader  func(h http.Header)
		logger       smartapi.Logger
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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
			api: func(api *smartapi.Server) {
				type foo struct {
					Name    string `json:"name"`
					Surname string `json:"surname"`
				}
				api.Post("/test", func(f *foo) error {
					require.Equal(t, "John", f.Name)
					require.Equal(t, "Smith", f.Surname)
					return nil
				},
					smartapi.JSONBody(foo{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "JSONBody Direct",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name": "John", "surname": "Smith"}`)))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Name    string `json:"name"`
					Surname string `json:"surname"`
				}
				api.Post("/test", func(f foo) error {
					require.Equal(t, "John", f.Name)
					require.Equal(t, "Smith", f.Surname)
					return nil
				},
					smartapi.JSONBodyDirect(foo{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "JSONBody Error",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name": "John", "surname": "Smith"`)))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Name    string `json:"name"`
					Surname string `json:"surname"`
				}
				api.Post("/test", func(f *foo) error {
					return nil
				},
					smartapi.JSONBody(foo{}),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte("{\"status\":400,\"reason\":\"cannot unmarshal request\"}\n"),
		},
		{
			name: "JSONBody Direct Error",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name": "John", "surname": "Smith"`)))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Name    string `json:"name"`
					Surname string `json:"surname"`
				}
				api.Post("/test", func(f foo) error {
					return nil
				},
					smartapi.JSONBodyDirect(foo{}),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte("{\"status\":400,\"reason\":\"cannot unmarshal request\"}\n"),
		},
		{
			name: "StringBody",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte("body value")))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(body string) error {
					require.Equal(t, "body value", body)
					return nil
				},
					smartapi.StringBody(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "StringBody Error",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", errorReader{})
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(body string) error {
					return nil
				},
					smartapi.StringBody(),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte("{\"status\":400,\"reason\":\"cannot read request\"}\n"),
		},
		{
			name: "ByteSliceBody",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte("body value")))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(body []byte) error {
					require.Equal(t, []byte("body value"), body)
					return nil
				},
					smartapi.ByteSliceBody(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "ByteSliceBody",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", errorReader{})
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(body []byte) error {
					return nil
				},
					smartapi.ByteSliceBody(),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte("{\"status\":400,\"reason\":\"cannot read request\"}\n"),
		},
		{
			name: "BodyReader",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte("body value")))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(body io.Reader) error {
					buff := make([]byte, 10)
					n, err := body.Read(buff)
					require.NoError(t, err)
					require.Equal(t, 10, n)
					require.Equal(t, []byte("body value"), buff)
					return nil
				},
					smartapi.BodyReader(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "ResponseWriter",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", bytes.NewReader([]byte("body value")))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter) {
					_, err := w.Write([]byte("RESPONSE"))
					require.NoError(t, err)
				},
					smartapi.ResponseWriter(),
				)
			},
			responseCode: http.StatusOK,
			responseBody: []byte("RESPONSE"),
		},
		{
			name: "Context",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(ctx context.Context) error {
					require.NotNil(t, ctx)
					return nil
				},
					smartapi.Context(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Full Request",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(r *http.Request) error {
					buff, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					require.Equal(t, `test`, string(buff))
					return nil
				},
					smartapi.Request(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Full Legacy 1",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					buff, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					w.WriteHeader(http.StatusMultiStatus)
					_, err = fmt.Fprintf(w, "response to %s!", buff)
					require.NoError(t, err)
				})
			},
			responseCode: http.StatusMultiStatus,
			responseBody: []byte("response to test!"),
		},
		{
			name: "Full Legacy 2",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					buff, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					w.WriteHeader(http.StatusMultiStatus)
					_, err = fmt.Fprintf(w, "response to %s!", buff)
					require.NoError(t, err)
				})
				api.Post("/test", f)
			},
			responseCode: http.StatusMultiStatus,
			responseBody: []byte("response to test!"),
		},
		{
			name: "Full Legacy 3",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					buff, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					w.WriteHeader(http.StatusMultiStatus)
					_, err = fmt.Fprintf(w, "response to %s!", buff)
					require.NoError(t, err)
				},
					smartapi.ResponseWriter(),
					smartapi.Request(),
				)
			},
			responseCode: http.StatusMultiStatus,
			responseBody: []byte("response to test!"),
		},
		{
			name: "Full Legacy With Error ok",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter, r *http.Request) error {
					buff, err := ioutil.ReadAll(r.Body)
					if err != nil {
						return err
					}
					_, err = fmt.Fprintf(w, "response to %s!", buff)
					if err != nil {
						return err
					}
					return nil
				},
					smartapi.ResponseWriter(),
					smartapi.Request(),
				)
			},
			responseCode: http.StatusOK,
			responseBody: []byte("response to test!"),
		},
		{
			name: "Legacy check fails with response code",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					buff, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					require.Equal(t, "test", string(buff))
				},
					smartapi.ResponseWriter(),
					smartapi.Request(),
					smartapi.ResponseStatus(http.StatusAccepted),
				)
			},
			responseCode: http.StatusAccepted,
			responseBody: nil,
		},
		{
			name: "Legacy check fails with wrong second argument",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("X-Test", "test")
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter, header string) {
					w.WriteHeader(http.StatusForbidden)
					_, err := fmt.Fprintf(w, "header value: %s!", header)
					require.NoError(t, err)
				},
					smartapi.ResponseWriter(),
					smartapi.Header("X-Test"),
				)
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte("header value: test!"),
		},
		{
			name: "Router",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/v1/foo/bar/test?test=test", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("X-Foo", "foo")
				return request
			},
			api: func(api *smartapi.Server) {
				api.Route("/v1/foo", func(r smartapi.Router) {
					r.Route("/bar", func(r smartapi.Router) {
						r.Get("/test", func(ctx context.Context, h string, qp string) {
							require.Equal(t, "foo", h)
							require.Equal(t, "test", qp)
						},
							smartapi.QueryParam("test"),
						)
					},
						smartapi.Header("X-Foo"),
					)
				},
					smartapi.Context(),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Tag Struct",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test/url?param=query&other_param=other_query&rparam=rquery", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.PostForm = url.Values{}
				request.PostForm.Set("post_param", "postquery")
				request.PostForm.Set("rpost_param", "rpostquery")
				request.Header.Set("X-Example", "header")
				request.Header.Set("X-Req-Example", "req_header")
				request.Header.Set("X-Another-Example", "other_header")
				request.Header.Set("Cookie", (&http.Cookie{Name: "User-Agent", Value: "Mozilla/5.0"}).String())
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Header          string              `smartapi:"header=X-Example"`
					RequiredHeader  string              `smartapi:"r_header=X-Req-Example"`
					Ctx             context.Context     `smartapi:"context"`
					URLParam        string              `smartapi:"url_param=param"`
					Fill            int                 ``
					QueryParam      string              `smartapi:"query_param=param"`
					RQueryParam     string              `smartapi:"r_query_param=rparam"`
					PostQueryParam  string              `smartapi:"post_query_param=post_param"`
					RPostQueryParam string              `smartapi:"r_post_query_param=rpost_param"`
					AnotherFill     int                 ``
					Cookie          string              `smartapi:"cookie=User-Agent"`
					Headers         smartapi.Headers    `smartapi:"response_headers"`
					Cookies         smartapi.Cookies    `smartapi:"response_cookies"`
					Response        http.ResponseWriter `smartapi:"response_writer"`
					RequestStruct   struct {
						AnotherHeader string `smartapi:"header=X-Another-Example"`
					} `smartapi:"request_struct"`
					RequestPtr *struct {
						AnotherQuery string `smartapi:"query_param=other_param"`
					} `smartapi:"request_struct"`
				}
				api.Post("/test/{param}", func(es *exampleStruct) {
					require.Equal(t, "header", es.Header)
					require.Equal(t, "req_header", es.RequiredHeader)
					require.NotNil(t, es.Ctx)
					require.Equal(t, "url", es.URLParam)
					require.Equal(t, "query", es.QueryParam)
					require.Equal(t, "rquery", es.RQueryParam)
					require.Equal(t, "postquery", es.PostQueryParam)
					require.Equal(t, "rpostquery", es.RPostQueryParam)
					require.Equal(t, "Mozilla/5.0", es.Cookie)
					require.Equal(t, "other_header", es.RequestStruct.AnotherHeader)
					require.Equal(t, "other_query", es.RequestPtr.AnotherQuery)
					es.Headers.Set("Response-Header", "header")
					es.Cookies.Add(&http.Cookie{Name: "Version", Value: "1.0"})
					es.Response.WriteHeader(http.StatusForbidden)
					_, err := fmt.Fprint(es.Response, "response")
					require.NoError(t, err)
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte("response"),
		},
		{
			name: "Tag Struct JSON Body",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader(`{"foo": "a", "bar": "b"}`))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type example struct {
					Foo string `json:"foo"`
					Bar string `json:"bar"`
				}
				type exampleStruct struct {
					Body example `smartapi:"json_body"`
				}
				api.Post("/test", func(es *exampleStruct) {
					require.Equal(t, "a", es.Body.Foo)
					require.Equal(t, "b", es.Body.Bar)
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Tag Struct String Body",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body string `smartapi:"string_body"`
				}
				api.Post("/test", func(es *exampleStruct) {
					require.Equal(t, "test", es.Body)
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Tag Struct Byte Slice Body",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body []byte `smartapi:"byte_slice_body"`
				}
				api.Post("/test", func(es *exampleStruct) {
					require.Equal(t, "test", string(es.Body))
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Tag Struct Body Reader",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body io.Reader `smartapi:"body_reader"`
				}
				api.Post("/test", func(es *exampleStruct) {
					all, err := ioutil.ReadAll(es.Body)
					require.NoError(t, err)
					require.Equal(t, "test", string(all))
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Tag Struct Full Request",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", strings.NewReader("test"))
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body *http.Request `smartapi:"request"`
				}
				api.Post("/test", func(es *exampleStruct) {
					all, err := ioutil.ReadAll(es.Body.Body)
					require.NoError(t, err)
					require.Equal(t, "test", string(all))
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Tag Struct Bad Request",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Foo string `smartapi:"r_header=X-Foo"`
				}
				api.Post("/test", func(es *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header X-Foo"}` + "\n"),
		},
		{
			name: "Tag Struct Direct Bad Request",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Foo string `smartapi:"r_header=X-Foo"`
				}
				api.Post("/test", func(es exampleStruct) {
				},
					smartapi.RequestStructDirect(exampleStruct{}),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header X-Foo"}` + "\n"),
		},
		{
			name: "Use Middleware",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Use(func(h http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						r.Header.Set("X-Middleware", "test")
						h.ServeHTTP(w, r)
					})
				})
				api.Post("/test", func(mhd string) error {
					require.Equal(t, "test", mhd)
					return nil
				},
					smartapi.Header("X-Middleware"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "With Middleware",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.With(func(h http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						r.Header.Set("X-Middleware", "test")
						h.ServeHTTP(w, r)
					})
				}).Post("/test", func(mhd string) error {
					require.Equal(t, "test", mhd)
					return nil
				},
					smartapi.Header("X-Middleware"),
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
			api: func(api *smartapi.Server) {
				api.Post("/test", func(test1, test2 string) error {
					require.Equal(t, "value", test1)
					require.Equal(t, "eulav", test2)
					return nil
				},
					smartapi.Header("X-Test1"),
					smartapi.Header("X-Test2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Required Header",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("X-Test1", "value")
				request.Header.Set("X-Test2", "eulav")
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(test1, test2 string) error {
					require.Equal(t, "value", test1)
					require.Equal(t, "eulav", test2)
					return nil
				},
					smartapi.RequiredHeader("X-Test1"),
					smartapi.RequiredHeader("X-Test2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Required Header Error",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(test1 string) error {
					return nil
				},
					smartapi.RequiredHeader("X-Test1"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header X-Test1"}` + "\n"),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(param1, param2 string) error {
					require.Equal(t, "eulav", param1)
					require.Equal(t, "value", param2)
					return nil
				},
					smartapi.QueryParam("param1"),
					smartapi.QueryParam("param2"),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(param1 string) error {
					return nil
				},
					smartapi.QueryParam("a"),
				)
			},
			logger: func() smartapi.Logger {
				m := mocks.NewMockLogger(ctrl)
				m.EXPECT().LogApiError(gomock.Any(), smartapi.Error(http.StatusBadRequest, "invalid URL escape \"%Z\"", "could not parse form")).Return().Times(1)
				return m
			}(),
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"could not parse form"}` + "\n"),
		},
		{
			name: "Required Query Params",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test?param2=value&param1=eulav", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(param1, param2 string) error {
					require.Equal(t, "eulav", param1)
					require.Equal(t, "value", param2)
					return nil
				},
					smartapi.RequiredQueryParam("param1"),
					smartapi.RequiredQueryParam("param2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Required Query Param Error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test?param2=value", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(param1, param2 string) error {
					return nil
				},
					smartapi.RequiredQueryParam("param1"),
					smartapi.RequiredQueryParam("param2"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte("{\"status\":400,\"reason\":\"missing required query param param1\"}\n"),
		},
		{
			name: "Post Query Params",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.PostForm = url.Values{}
				request.PostForm.Set("param1", "eulav")
				request.PostForm.Set("param2", "value")
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func(param1, param2 string) error {
					require.Equal(t, "eulav", param1)
					require.Equal(t, "value", param2)
					return nil
				},
					smartapi.PostQueryParam("param1"),
					smartapi.PostQueryParam("param2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Required Post Query Params",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.PostForm = url.Values{}
				request.PostForm.Set("param1", "eulav")
				request.PostForm.Set("param2", "value")
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(param1, param2 string) error {
					require.Equal(t, "eulav", param1)
					require.Equal(t, "value", param2)
					return nil
				},
					smartapi.RequiredPostQueryParam("param1"),
					smartapi.RequiredPostQueryParam("param2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Required Query Param Error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				request.PostForm = url.Values{}
				request.PostForm.Set("param2", "value")
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(param1, param2 string) error {
					return nil
				},
					smartapi.RequiredPostQueryParam("param1"),
					smartapi.RequiredPostQueryParam("param2"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte("{\"status\":400,\"reason\":\"missing required post query param param1\"}\n"),
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
			api: func(api *smartapi.Server) {
				api.Get("/test/{param1}/orders/{param2}", func(param1, param2 string) error {
					require.Equal(t, "foo", param1)
					require.Equal(t, "bar", param2)
					return nil
				},
					smartapi.URLParam("param1"),
					smartapi.URLParam("param2"),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(c1, c2 string) error {
					require.Equal(t, "foo", c1)
					require.Equal(t, "bar", c2)
					return nil
				},
					smartapi.Cookie("Test1"),
					smartapi.Cookie("Test2"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Empty Cookie",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(c1 string) error {
					require.Equal(t, "", c1)
					return nil
				},
					smartapi.Cookie("Test1"),
				)
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Required Cookies",
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(c1, c2 string) error {
					require.Equal(t, "foo", c1)
					require.Equal(t, "bar", c2)
					return nil
				},
					smartapi.RequiredCookie("Test1"),
					smartapi.RequiredCookie("Test2"),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(cookie string) error {
					return nil
				},
					smartapi.RequiredCookie("Test1"),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(headers smartapi.Headers) error {
					headers.Set("Test1", "foo")
					headers.Set("Test2", "bar")
					return nil
				},
					smartapi.ResponseHeaders(),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func(cookies smartapi.Cookies) error {
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
					smartapi.ResponseCookies(),
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
			api := smartapi.NewServer(tt.logger)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, tt.responseCode, r.Code)
			require.Equal(t, bytes.NewBuffer(tt.responseBody), r.Body)

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
		api          func(api *smartapi.Server)
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
			api: func(api *smartapi.Server) {
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return nil
				},
					smartapi.ResponseStatus(http.StatusAccepted),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (string, error) {
					return "foobar", nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte("foobar"),
		},
		{
			name: "String Handler bad request",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(header string) (string, error) {
					return "foobar", nil
				},
					smartapi.RequiredHeader("Some-Header"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header Some-Header"}` + "\n"),
		},
		{
			name: "String Handler error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (string, error) {
					return "", smartapi.Error(http.StatusForbidden, "forbidden", "forbidden")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"forbidden"}` + "\n"),
		},
		{
			name: "String Handler no content",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (string, error) {
					return "", nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func() ([]byte, error) {
					return []byte{1, 2, 45, 23}, nil
				})
			},
			responseCode: http.StatusOK,
			responseBody: []byte{1, 2, 45, 23},
		},
		{

			name: "Byte Slice Handler Bad Request",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func(header string) ([]byte, error) {
					return []byte{1, 2, 45, 23}, nil
				},
					smartapi.RequiredHeader("Some-Header"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header Some-Header"}` + "\n"),
		},
		{

			name: "Byte Slice Handler Error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func() ([]byte, error) {
					return nil, smartapi.Error(http.StatusForbidden, "forbidden", "forbidden")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"forbidden"}` + "\n"),
		},
		{

			name: "Byte Slice No Content",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func() ([]byte, error) {
					return nil, nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
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
			api: func(api *smartapi.Server) {
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
			name: "Struct handler bad request",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				api.Get("/test", func(h string) (foo, error) {
					return foo{
						Field1: "foo",
						Field2: "bar",
					}, nil
				},
					smartapi.RequiredHeader("Some-Header"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header Some-Header"}` + "\n"),
		},
		{
			name: "Struct handler error result",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				api.Get("/test", func() (foo, error) {
					return foo{}, smartapi.Error(http.StatusForbidden, "forbidden", "forbidden")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"forbidden"}` + "\n"),
		},
		{
			name: "Struct handler marshal error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string   `json:"field1"`
					Field2 chan int `json:"\x00"`
				}
				api.Get("/test", func() (foo, error) {
					return foo{
						Field1: "test",
						Field2: make(chan int),
					}, nil
				})
			},
			responseCode: http.StatusInternalServerError,
			responseBody: []byte(`{"status":500,"reason":"cannot encode response"}` + "\n"),
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
			api: func(api *smartapi.Server) {
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
			name: "Pointer handler bad request",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				api.Get("/test", func(h string) (*foo, error) {
					return &foo{
						Field1: "foo",
						Field2: "bar",
					}, nil
				},
					smartapi.RequiredHeader("Some-Header"),
				)
			},
			responseCode: http.StatusBadRequest,
			responseBody: []byte(`{"status":400,"reason":"missing required header Some-Header"}` + "\n"),
		},
		{
			name: "Pointer handler error result",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				api.Get("/test", func() (*foo, error) {
					return nil, smartapi.Error(http.StatusForbidden, "forbidden", "forbidden")
				})
			},
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"forbidden"}` + "\n"),
		},
		{
			name: "Pointer handler no result",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string `json:"field1"`
					Field2 string `json:"field2"`
				}
				api.Get("/test", func() (*foo, error) {
					return nil, nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "Pointer handler marshal error",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				type foo struct {
					Field1 string   `json:"field1"`
					Field2 chan int `json:"\x00"`
				}
				api.Get("/test", func() (*foo, error) {
					return &foo{
						Field1: "test",
						Field2: make(chan int),
					}, nil
				})
			},
			responseCode: http.StatusInternalServerError,
			responseBody: []byte(`{"status":500,"reason":"cannot encode response"}` + "\n"),
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
			api: func(api *smartapi.Server) {
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
			api: func(api *smartapi.Server) {
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
			api := smartapi.NewServer(nil)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, tt.responseCode, r.Code)
			require.Equal(t, bytes.NewBuffer(tt.responseBody), r.Body)
		})
	}
}

func TestHandlersErrors(t *testing.T) {
	type test struct {
		name   string
		api    func(api *smartapi.Server)
		expect error
	}

	tests := []test{
		{
			name: "Nil handler",
			api: func(api *smartapi.Server) {
				api.Get("/test", nil)
			},
			expect: errors.New("endpoint /test: nil handler"),
		},
		{
			name: "Too many arguments",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value string) error {
					return nil
				})
			},
			expect: errors.New("endpoint /test: number of arguments of a function doesn't match provided arguments"),
		},
		{
			name: "Too little arguments",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return nil
				},
					smartapi.QueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: number of arguments of a function doesn't match provided arguments"),
		},
		{
			name: "Non function handler",
			api: func(api *smartapi.Server) {
				api.Get("/test", 456)
			},
			expect: errors.New("endpoint /test: handler must be a function"),
		},
		{
			name: "Only one read argument at a time",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(str string, bts []byte) (string, error) {
					return "", nil
				},
					smartapi.StringBody(),
					smartapi.ByteSliceBody(),
				)
			},
			expect: errors.New("endpoint /test: only one argument can read request's body"),
		},
		{
			name: "Many errors at once",
			api: func(api *smartapi.Server) {
				api.Get("/test", 456)
				api.Get("/foo", "hello")
				api.Get("/bar", []string{"shit"})
			},
			expect: errors.New("endpoint /test: handler must be a function, endpoint /foo: handler must be a function, endpoint /bar: handler must be a function"),
		},
		{
			name: "Invalid return type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() int {
					return 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 2",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (string, int) {
					return "", 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 3",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (struct{}, int) {
					return struct{}{}, 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 4",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (*struct{}, int) {
					return &struct{}{}, 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "Invalid return type 5",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() ([]byte, int) {
					return []byte(""), 0
				})
			},
			expect: errors.New("endpoint /test: expect an error type in return arguments"),
		},
		{
			name: "QueryParam wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.QueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "Required QueryParam wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.RequiredQueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "PostQueryParam wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.PostQueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "Required PostQueryParam wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.RequiredPostQueryParam("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "URLParam wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test/{name}", func(value int) error {
					return nil
				},
					smartapi.URLParam("name"),
				)
			},
			expect: errors.New("endpoint /test/{name}: (argument 0) expected a string type"),
		},
		{
			name: "Header wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.Header("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "Tag Struct Error",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Value string `smartapi:"header=X-Example"`
				}
				api.Post("/test", func(es *exampleStruct) {
				},
					smartapi.RequestStruct(23),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) RequestStruct's argument must be a structure"),
		},
		{
			name: "Full Request Wrong Type",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(r int) error {
					return nil
				},
					smartapi.Request(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) argument's type must be *http.Request"),
		},
		{
			name: "Legacy check fails with response code",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					buff, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					require.Equal(t, "test", string(buff))
				},
					smartapi.ResponseStatus(http.StatusAccepted),
				)
			},
			expect: errors.New("endpoint /test: number of arguments of a function doesn't match provided arguments"),
		},
		{
			name: "Required header wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.RequiredHeader("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "Cookie wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.Cookie("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "Required Cookie wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.RequiredCookie("name"),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "JSON body wrong type",
			api: func(api *smartapi.Server) {
				type s struct {
					Field string
				}
				api.Get("/test", func(value s) error {
					return nil
				},
					smartapi.JSONBody(s{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) invalid type"),
		},
		{
			name: "JSON body wrong type",
			api: func(api *smartapi.Server) {
				type s struct {
					Field string
				}
				api.Get("/test", func(value *s) error {
					return nil
				},
					smartapi.JSONBodyDirect(s{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) invalid type"),
		},
		{
			name: "String body wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.StringBody(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected string type"),
		},
		{
			name: "Byte slice wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value int) error {
					return nil
				},
					smartapi.ByteSliceBody(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected a byte slice"),
		},
		{
			name: "Reader wrong type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func(value interface{}) error {
					return nil
				},
					smartapi.BodyReader(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected io.Reader interface"),
		},
		{
			name: "Context Wrong Type",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(ctx int) error {
					return nil
				},
					smartapi.Context(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) expected context.Context"),
		},
		{
			name: "Headers Wrong Type",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(test int) error {
					return nil
				},
					smartapi.ResponseHeaders(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) argument's type must be smartapi.Headers"),
		},
		{
			name: "Cookies Wrong Type",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(test int) error {
					return nil
				},
					smartapi.ResponseCookies(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) argument's type must be smartapi.Cookies"),
		},
		{
			name: "Response Writer Wrong Type",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(test int) {
					return
				},
					smartapi.ResponseWriter(),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) argument's type must be http.ResponseWriter"),
		},
		{
			name: "Response Writer Cannot return response",
			api: func(api *smartapi.Server) {
				api.Post("/test", func(w http.ResponseWriter) (string, error) {
					_, _ = w.Write([]byte("RESPONSE"))
					return "string response", nil
				},
					smartapi.ResponseWriter(),
				)
			},
			expect: errors.New("endpoint /test: cannot write response and return response"),
		},
		{
			name: "Invalid return value type",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (func(), error) {
					return func() {
					}, nil
				})
			},
			expect: errors.New("endpoint /test: unsupported return type"),
		},
		{
			name: "Too many return arguments",
			api: func(api *smartapi.Server) {
				api.Get("/test", func() (string, string, error) {
					return "", "", nil
				})
			},
			expect: errors.New("endpoint /test: invalid number of return arguments"),
		},
		{
			name: "Tag Struct Non Pointer Type",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body string `smartapi:"string_body"`
				}
				api.Post("/test", func(s int) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) argument must be a pointer"),
		},
		{
			name: "Tag Struct Wrong Pointer Type",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body string `smartapi:"string_body"`
				}
				api.Post("/test", func(s *smartapi.Server) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) invalid argument type"),
		},
		{
			name: "Tag Struct Direct Pointer Type",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body string `smartapi:"string_body"`
				}
				api.Post("/test", func(s *smartapi.Server) {
				},
					smartapi.RequestStructDirect(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) invalid argument type"),
		},
		{
			name: "Tag Struct Wrong Field Type",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body int `smartapi:"string_body"`
				}
				api.Post("/test", func(s *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) (struct field Body) expected string type"),
		},
		{
			name: "Tag Struct Invalid Tag",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Body int `smartapi:"non_existing_tag"`
				}
				api.Post("/test", func(s *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) (struct field Body) unsupported tag"),
		},
		{
			name: "Tag Struct Direct Not A Struct",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Inner int `smartapi:"request_struct"`
				}
				api.Post("/test", func(s *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) (struct field Inner) invalid type of request_struct"),
		},
		{
			name: "Tag Struct Ptr To A Non-Struct",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Inner *int `smartapi:"request_struct"`
				}
				api.Post("/test", func(s *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) (struct field Inner) RequestStruct's argument must be a structure"),
		},
		{
			name: "Tag Struct Inner request struct error",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					Inner struct {
						Header int `smartapi:"header=something"`
					} `smartapi:"request_struct"`
				}
				api.Post("/test", func(s *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) (struct field Inner) (struct field Header) expected a string type"),
		},
		{
			name: "Tag Struct Multiple Readers",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					StrBody  string `smartapi:"string_body"`
					ByteBody []byte `smartapi:"byte_slice_body"`
				}
				api.Post("/test", func(s *exampleStruct) {
				},
					smartapi.RequestStruct(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) only one struct field can read request's body"),
		},
		{
			name: "Tag Struct Multiple Readers Direct",
			api: func(api *smartapi.Server) {
				type exampleStruct struct {
					StrBody  string `smartapi:"string_body"`
					ByteBody []byte `smartapi:"byte_slice_body"`
				}
				api.Post("/test", func(s exampleStruct) {
				},
					smartapi.RequestStructDirect(exampleStruct{}),
				)
			},
			expect: errors.New("endpoint /test: (argument 0) only one struct field can read request's body"),
		},
		{
			name: "Router Pass Error",
			api: func(api *smartapi.Server) {
				api.Route("/v1/user", func(r smartapi.Router) {
					r.Get("/test", func(qp int) {
						require.Equal(t, "test", qp)
					},
						smartapi.QueryParam("test"),
					)
				})
			},
			expect: errors.New("route /v1/user: endpoint /test: (argument 0) expected a string type"),
		},
		{
			name: "Router Pass Error",
			api: func(api *smartapi.Server) {
				api.Route("/v1/user", nil)
			},
			expect: errors.New("route /v1/user: nil handler"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := smartapi.NewServer(nil)
			tt.api(api)
			_, err := api.Handler()
			require.Equal(t, tt.expect, err)
		})
	}
}

func TestMethods(t *testing.T) {
	type test struct {
		name         string
		request      func() *http.Request
		api          func(api *smartapi.Server)
		responseCode int
		responseBody []byte
		logger       smartapi.Logger
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []test{
		{
			name: "POST",
			request: func() *http.Request {
				request, err := http.NewRequest("POST", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Post("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "GET",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "PUT",
			request: func() *http.Request {
				request, err := http.NewRequest("PUT", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Put("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "PATCH",
			request: func() *http.Request {
				request, err := http.NewRequest("PATCH", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Patch("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "DELETE",
			request: func() *http.Request {
				request, err := http.NewRequest("DELETE", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Delete("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "HEAD",
			request: func() *http.Request {
				request, err := http.NewRequest("HEAD", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Head("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "OPTIONS",
			request: func() *http.Request {
				request, err := http.NewRequest("OPTIONS", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Options("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "CONNECT",
			request: func() *http.Request {
				request, err := http.NewRequest("CONNECT", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Connect("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
		{
			name: "TRACE",
			request: func() *http.Request {
				request, err := http.NewRequest("TRACE", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Trace("/test", func() error {
					return nil
				})
			},
			responseCode: http.StatusNoContent,
			responseBody: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request()
			api := smartapi.NewServer(tt.logger)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, tt.responseCode, r.Code)
			require.Equal(t, bytes.NewBuffer(tt.responseBody), r.Body)
		})
	}
}

func TestError(t *testing.T) {
	type test struct {
		name         string
		request      func() *http.Request
		api          func(api *smartapi.Server)
		responseCode int
		responseBody []byte
		logger       smartapi.Logger
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return smartapi.Error(http.StatusForbidden, "message", "reason")
				})
			},
			logger: func() smartapi.Logger {
				m := mocks.NewMockLogger(ctrl)
				m.EXPECT().LogApiError(gomock.Any(), smartapi.Error(http.StatusForbidden, "message", "reason")).Return().Times(1)
				return m
			}(),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return smartapi.Errorf(http.StatusForbidden, "message: %s!", "format")
				})
			},
			logger: func() smartapi.Logger {
				m := mocks.NewMockLogger(ctrl)
				m.EXPECT().LogApiError(gomock.Any(), smartapi.Error(http.StatusForbidden, "message: format!", "unknown")).Do(
					func(ctx context.Context, err smartapi.ApiError) {
						require.Equal(t, err.Error(), "message: format!")
					},
				).Times(1)
				return m
			}(),
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
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return smartapi.WrapError(http.StatusForbidden, errors.New("error"), "reason")
				})
			},
			logger: func() smartapi.Logger {
				m := mocks.NewMockLogger(ctrl)
				m.EXPECT().LogApiError(gomock.Any(), smartapi.Error(http.StatusForbidden, "error", "reason")).Return().Times(1)
				return m
			}(),
			responseCode: http.StatusForbidden,
			responseBody: []byte(`{"status":403,"reason":"reason"}` + "\n"),
		},
		{
			name: "OrdinaryError",
			request: func() *http.Request {
				request, err := http.NewRequest("GET", "/test", nil)
				if err != nil {
					t.Fatal(err)
				}
				return request
			},
			api: func(api *smartapi.Server) {
				api.Get("/test", func() error {
					return errors.New("error")
				})
			},
			logger: func() smartapi.Logger {
				m := mocks.NewMockLogger(ctrl)
				m.EXPECT().LogError(gomock.Any(), errors.New("error")).Return().Times(1)
				return m
			}(),
			responseCode: http.StatusInternalServerError,
			responseBody: []byte(`{"status":500,"reason":"unknown"}` + "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request()
			api := smartapi.NewServer(tt.logger)
			tt.api(api)

			r := httptest.NewRecorder()

			handler, err := api.Handler()
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(r, request)

			require.Equal(t, tt.responseCode, r.Code)
			require.Equal(t, bytes.NewBuffer(tt.responseBody), r.Body)
		})
	}
}

func TestStartAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		a       smartapi.API
		address string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "StartOK",
			args: args{
				a: func() smartapi.API {
					m := mocks.NewMockAPI(ctrl)
					m.EXPECT().Init().Times(1)
					m.EXPECT().Start(":80").Return(nil).Times(1)
					return m
				}(),
				address: ":80",
			},
			wantErr: false,
		},
		{
			name: "StartError",
			args: args{
				a: func() smartapi.API {
					m := mocks.NewMockAPI(ctrl)
					m.EXPECT().Init().Times(1)
					m.EXPECT().Start(":80").Return(errors.New("some error")).Times(1)
					return m
				}(),
				address: ":80",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := smartapi.StartAPI(tt.args.a, tt.args.address); (err != nil) != tt.wantErr {
				t.Errorf("StartAPI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
