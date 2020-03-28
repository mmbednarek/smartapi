package smartapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
)

type endpointOptions int

const (
	flagArgument endpointOptions = 1 << iota
	flagParsesQuery
	flagResponseStatus
	flagMiddleware
	flagReadsRequestBody
)

func (e endpointOptions) has(o endpointOptions) bool {
	return e & o != 0
}

// EndpointParam is used with endpoint definition
type EndpointParam interface {
	options() endpointOptions
}

// Argument represents an argument passed to a function
type Argument interface {
	checkArg(arg reflect.Type) error
	getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error)
}

type headerArgument struct {
	name string
}

func (a headerArgument) options() endpointOptions {
	return flagArgument
}

func (a headerArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Header.Get(a.name)), nil
}

func (a headerArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

// Header reads a header from the request and passes it as string to a function
func Header(name string) EndpointParam {
	return headerArgument{name: name}
}

type requiredHeaderArgument struct {
	name string
}

func (a requiredHeaderArgument) options() endpointOptions {
	return flagArgument
}

func (a requiredHeaderArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	value := r.Header.Get(a.name)
	if len(value) == 0 {
		msg := fmt.Sprintf("missing required header %s", a.name)
		return reflect.Value{}, Error(http.StatusBadRequest, msg, msg)
	}
	return reflect.ValueOf(value), nil
}

func (a requiredHeaderArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

// RequiredHeader reads a header from the request and passes it as string to a function
func RequiredHeader(name string) EndpointParam {
	return requiredHeaderArgument{name: name}
}

type jsonBodyArgument struct {
	typ reflect.Type
}

func (a jsonBodyArgument) options() endpointOptions {
	return flagArgument | flagReadsRequestBody
}

func (a jsonBodyArgument) checkArg(arg reflect.Type) error {
	if reflect.PtrTo(a.typ) != arg {
		return errors.New("invalid type")
	}
	return nil
}

func (a jsonBodyArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	value := reflect.New(a.typ)
	obj := value.Interface()
	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return reflect.Value{}, WrapError(http.StatusBadRequest, err, "cannot unmarshal request")
	}
	return value, nil
}

// JSONBody reads request's body and unmarshals it into a json structure
func JSONBody(v interface{}) EndpointParam {
	return jsonBodyArgument{typ: reflect.TypeOf(v)}
}

type stringBodyArgument struct{}

func (stringBodyArgument) options() endpointOptions {
	return flagArgument | flagReadsRequestBody
}

func (s stringBodyArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected string type")
	}
	return nil
}

func (s stringBodyArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	result, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return reflect.Value{}, WrapError(http.StatusBadRequest, err, "cannot read request")
	}
	return reflect.ValueOf(string(result)), nil
}

// StringBody reads request's body end passes it as a string
func StringBody() EndpointParam {
	return stringBodyArgument{}
}

type byteSliceBodyArgument struct{}

var byteSliceType = reflect.TypeOf([]byte(nil))

func (byteSliceBodyArgument) options() endpointOptions {
	return flagArgument | flagReadsRequestBody
}

func (s byteSliceBodyArgument) checkArg(arg reflect.Type) error {
	if arg != byteSliceType {
		return errors.New("expected a byte slice")
	}
	return nil
}

func (s byteSliceBodyArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	result, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return reflect.Value{}, WrapError(http.StatusBadRequest, err, "cannot read request")
	}
	return reflect.ValueOf(result), nil
}

// ByteSliceBody reads request's body end passes it as a byte slice.
func ByteSliceBody() EndpointParam {
	return byteSliceBodyArgument{}
}

type bodyReaderArgument struct{}

var readerType = reflect.TypeOf((*io.Reader)(nil)).Elem()

func (bodyReaderArgument) options() endpointOptions {
	return flagArgument | flagReadsRequestBody
}

func (b bodyReaderArgument) checkArg(arg reflect.Type) error {
	if arg != readerType {
		return errors.New("expected io.Reader interface")
	}
	return nil
}

func (b bodyReaderArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Body), nil
}

// BodyReader passes an io.Reader interface to read request's body.
func BodyReader() EndpointParam {
	return bodyReaderArgument{}
}

type urlParamArgument struct {
	name string
}

func (urlParamArgument) options() endpointOptions {
	return flagArgument
}

func (u urlParamArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (u urlParamArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(chi.URLParam(r, u.name)), nil
}

// URLParam reads a url param and passes it as a string
func URLParam(name string) EndpointParam {
	return urlParamArgument{name: name}
}

type contextArgument struct {
}

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

func (contextArgument) options() endpointOptions {
	return flagArgument
}

func (q contextArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.Interface || !arg.Implements(ctxType) {
		return errors.New("expected context.Context")
	}
	return nil
}

func (q contextArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Context()), nil
}

// Context passes request's context into the function
func Context() EndpointParam {
	return contextArgument{}
}

type responseStatusArgument struct {
	status int
}

func (responseStatusArgument) options() endpointOptions {
	return flagResponseStatus
}

// ResponseStatus allows to set successful response status
func ResponseStatus(status int) EndpointParam {
	return responseStatusArgument{status: status}
}

type queryParamArgument struct {
	name string
}

func (queryParamArgument) options() endpointOptions {
	return flagArgument | flagParsesQuery
}

func (q queryParamArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (q queryParamArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Form.Get(q.name)), nil
}

// QueryParam reads a query param and passes it as a string
func QueryParam(name string) EndpointParam {
	return queryParamArgument{name: name}
}

type postQueryParamArgument struct {
	name string
}

func (postQueryParamArgument) options() endpointOptions {
	return flagArgument | flagParsesQuery
}

func (p postQueryParamArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (p postQueryParamArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.PostForm.Get(p.name)), nil
}

// PostQueryParam parses query end passes post query param into a string as an argument
func PostQueryParam(name string) EndpointParam {
	return postQueryParamArgument{name: name}
}

type cookieArgument struct {
	name string
}

func (cookieArgument) options() endpointOptions {
	return flagArgument
}

func (c cookieArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (c cookieArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		msg := fmt.Sprintf("missing cookie %s", c.name)
		return reflect.Value{}, Error(http.StatusBadRequest, msg, msg)
	}
	return reflect.ValueOf(cookie.Value), nil
}

// Cookie reads a cookie from the request and passes it as a string
func Cookie(name string) EndpointParam {
	return cookieArgument{name: name}
}

type headerSetterArgument struct{}

var headerSetterType = reflect.TypeOf((*Headers)(nil)).Elem()

func (headerSetterArgument) options() endpointOptions {
	return flagArgument
}

func (headerSetterArgument) checkArg(arg reflect.Type) error {
	if arg != headerSetterType {
		return errors.New("argument's type must be smartapi.Headers")
	}
	return nil
}

func (headerSetterArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(w.Header()), nil
}

// ResponseHeaders passes an interface to set response header values
func ResponseHeaders() EndpointParam {
	return headerSetterArgument{}
}

type cookieSetterArgument struct{}

var cookieSetterType = reflect.TypeOf((*Cookies)(nil)).Elem()

func (cookieSetterArgument) options() endpointOptions {
	return flagArgument
}

func (c cookieSetterArgument) checkArg(arg reflect.Type) error {
	if arg != cookieSetterType {
		return errors.New("argument's type must be smartapi.Cookies")
	}
	return nil
}

func (c cookieSetterArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(cookieSetter{w: w}), nil
}

// ResponseCookies passes an interface to set cookie values
func ResponseCookies() EndpointParam {
	return cookieSetterArgument{}
}

type middleware struct {
	middlewares []func(http.Handler) http.Handler
}

func (middleware) options() endpointOptions {
	return flagMiddleware
}

// Middleware allows to use chi middlewares
func Middleware(m ...func(http.Handler) http.Handler) EndpointParam {
	return middleware{middlewares: m}
}
