package smartapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
)

type headerArgument struct {
	name string
}

func (a headerArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Header.Get(a.name)), nil
}

func (a headerArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string")
	}
	return nil
}

func Header(name string) Argument {
	return headerArgument{name: name}
}

type jsonBodyArgument struct {
	typ reflect.Type
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
		return reflect.Value{}, err
	}
	return value, nil
}

func JSONBody(v interface{}) Argument {
	return jsonBodyArgument{typ: reflect.TypeOf(v)}
}

type urlParamArgument struct {
	name string
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

func URLParam(name string) Argument {
	return urlParamArgument{name: name}
}

type contextArgument struct {
}

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

func (q contextArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.Interface || !arg.Implements(ctxType) {
		return errors.New("expected context.Context")
	}
	return nil
}

func (q contextArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Context()), nil
}

func Context() Argument {
	return contextArgument{}
}

type returnStatusArgument struct {
	status int
}

func (a returnStatusArgument) checkArg(arg reflect.Type) error {
	return nil
}

func (a returnStatusArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.Value{}, nil
}

func ReturnStatus(status int) Argument {
	return returnStatusArgument{status: status}
}

type queryParamArgument struct {
	name string
}

func (q queryParamArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string")
	}
	return nil
}

func (q queryParamArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r.Form.Get(q.name)), nil
}

func QueryParam(name string) Argument {
	return queryParamArgument{name: name}
}

type cookieArgument struct {
	name string
}

var cookieType = reflect.TypeOf((*http.Cookie)(nil))

func (c cookieArgument) checkArg(arg reflect.Type) error {
	if arg != cookieType {
		return errors.New("expected a cookie type")
	}
	return nil
}

func (c cookieArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(cookie), nil
}

func Cookie(name string) Argument {
	return cookieArgument{name: name}
}

type cookieValueArgument struct {
	name string
}

func (c cookieValueArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string")
	}
	return nil
}

func (c cookieValueArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		msg := fmt.Sprintf("missing cookie %s", c.name)
		return reflect.Value{}, Error(http.StatusBadRequest, msg, msg)
	}
	return reflect.ValueOf(cookie.Value), nil
}

func CookieValue(name string) Argument {
	return cookieValueArgument{name: name}
}

type headerSetterArgument struct{}

var headerSetterType = reflect.TypeOf((*Headers)(nil)).Elem()

func (headerSetterArgument) checkArg(arg reflect.Type) error {
	if arg != headerSetterType {
		return errors.New("argument's type must be smartapi.Headers")
	}
	return nil
}

func (headerSetterArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(w.Header()), nil
}

func ResponseHeaders() Argument {
	return headerSetterArgument{}
}

type cookieSetterArgument struct{}

var cookieSetterType = reflect.TypeOf((*Cookies)(nil)).Elem()

func (c cookieSetterArgument) checkArg(arg reflect.Type) error {
	if arg != cookieSetterType {
		return errors.New("argument's type must be smartapi.Cookies")
	}
	return nil
}

func (c cookieSetterArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(cookieSetter{w: w}), nil
}

func ResponseCookies() Argument {
	return cookieSetterArgument{}
}

type middlewareArgument struct {
	middlewares []func(http.Handler) http.Handler
}

func (m middlewareArgument) checkArg(arg reflect.Type) error {
	return nil
}

func (m middlewareArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.Value{}, nil
}

func Middleware(m ...func(http.Handler) http.Handler) Argument {
	return middlewareArgument{middlewares: m}
}
