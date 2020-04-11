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
	"strconv"

	"github.com/go-chi/chi"
)

type endpointOptions int

const (
	flagArgument endpointOptions = 1 << iota
	flagParsesQuery
	flagResponseStatus
	flagReadsRequestBody
	flagWritesResponse
	flagError
)

func (e endpointOptions) has(o endpointOptions) bool {
	return e&o != 0
}

// EndpointParam is used with endpointData definition
type EndpointParam interface {
	options() endpointOptions
}

// Argument represents an argument passed to a function
type Argument interface {
	EndpointParam
	checkArg(arg reflect.Type) error
	getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error)
}

type errorEndpointParam struct {
	err error
}

func (e errorEndpointParam) options() endpointOptions {
	return flagError
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

// JSONBody reads request's body and unmarshals it into a pointer to a json structure
func JSONBody(v interface{}) EndpointParam {
	return jsonBodyArgument{typ: reflect.TypeOf(v)}
}

type jsonBodyDirectArgument struct {
	typ reflect.Type
}

func (a jsonBodyDirectArgument) options() endpointOptions {
	return flagArgument | flagReadsRequestBody
}

func (a jsonBodyDirectArgument) checkArg(arg reflect.Type) error {
	if a.typ != arg {
		return errors.New("invalid type")
	}
	return nil
}

func (a jsonBodyDirectArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	value := reflect.New(a.typ)
	obj := value.Interface()
	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return reflect.Value{}, WrapError(http.StatusBadRequest, err, "cannot unmarshal request")
	}
	return value.Elem(), nil
}

// JSONBodyDirect reads request's body and unmarshals it into a json structure
func JSONBodyDirect(v interface{}) EndpointParam {
	return jsonBodyDirectArgument{typ: reflect.TypeOf(v)}
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

type requiredQueryParamArgument struct {
	name string
}

func (requiredQueryParamArgument) options() endpointOptions {
	return flagArgument | flagParsesQuery
}

func (q requiredQueryParamArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (q requiredQueryParamArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	value := r.Form.Get(q.name)
	if len(value) == 0 {
		m := fmt.Sprintf("missing required query param %s", q.name)
		return reflect.Value{}, Error(http.StatusBadRequest, m, m)
	}
	return reflect.ValueOf(value), nil
}

// RequiredQueryParam reads a query param and passes it as a string. Returns 400 BAD REQUEST when empty
func RequiredQueryParam(name string) EndpointParam {
	return requiredQueryParamArgument{name: name}
}

type requiredPostQueryParamArgument struct {
	name string
}

func (requiredPostQueryParamArgument) options() endpointOptions {
	return flagArgument | flagParsesQuery
}

func (q requiredPostQueryParamArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (q requiredPostQueryParamArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	value := r.PostForm.Get(q.name)
	if len(value) == 0 {
		m := fmt.Sprintf("missing required post query param %s", q.name)
		return reflect.Value{}, Error(http.StatusBadRequest, m, m)
	}
	return reflect.ValueOf(value), nil
}

// RequiredPostQueryParam reads a post query param and passes it as a string. Returns 400 BAD REQUEST if empty.
func RequiredPostQueryParam(name string) EndpointParam {
	return requiredPostQueryParamArgument{name: name}
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
		return reflect.ValueOf(""), nil
	}
	return reflect.ValueOf(cookie.Value), nil
}

// Cookie reads a cookie from the request and passes it as a string
func Cookie(name string) EndpointParam {
	return cookieArgument{name: name}
}

type requiredCookieArgument struct {
	name string
}

func (requiredCookieArgument) options() endpointOptions {
	return flagArgument
}

func (c requiredCookieArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.String {
		return errors.New("expected a string type")
	}
	return nil
}

func (c requiredCookieArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		msg := fmt.Sprintf("missing cookie %s", c.name)
		return reflect.Value{}, Error(http.StatusBadRequest, msg, msg)
	}
	return reflect.ValueOf(cookie.Value), nil
}

// RequiredCookie reads a cookie from the request and passes it as a string
func RequiredCookie(name string) EndpointParam {
	return requiredCookieArgument{name: name}
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

type responseWriterArgument struct{}

var responseWriterType = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()

func (responseWriterArgument) checkArg(arg reflect.Type) error {
	if arg != responseWriterType {
		return errors.New("argument's type must be http.ResponseWriter")
	}
	return nil
}

func (responseWriterArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(w), nil
}

func (responseWriterArgument) options() endpointOptions {
	return flagArgument | flagWritesResponse
}

func ResponseWriter() EndpointParam {
	return responseWriterArgument{}
}

type fullRequestArgument struct{}

var fullRequestType = reflect.TypeOf(&http.Request{})

func (fullRequestArgument) options() endpointOptions {
	return flagArgument | flagReadsRequestBody
}

func (fullRequestArgument) checkArg(arg reflect.Type) error {
	if arg != fullRequestType {
		return errors.New("argument's type must be *http.Request")
	}
	return nil
}

func (fullRequestArgument) getValue(_ http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return reflect.ValueOf(r), nil
}

func Request() EndpointParam {
	return fullRequestArgument{}
}

const smartAPITagName = "smartapi"

type tagStructArgument struct {
	structType reflect.Type
	flags      endpointOptions
	arguments  []Argument
}

func (t tagStructArgument) options() endpointOptions {
	return t.flags
}

func (t tagStructArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.Ptr {
		return errors.New("argument must be a pointer")
	}
	if t.structType != arg.Elem() {
		return errors.New("invalid argument type")
	}
	return nil
}

func (t tagStructArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	return constructStruct(t.structType, t.arguments, w, r)
}

type tagStructDirectArgument tagStructArgument

func (t tagStructDirectArgument) options() endpointOptions {
	return t.flags
}

func (t tagStructDirectArgument) checkArg(arg reflect.Type) error {
	if t.structType != arg {
		return errors.New("invalid argument type")
	}
	return nil
}

func (t tagStructDirectArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	v, err := constructStruct(t.structType, t.arguments, w, r)
	if err != nil {
		return reflect.Value{}, err
	}
	return v.Elem(), nil
}

func constructStruct(structType reflect.Type, args []Argument, w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	vPtr := reflect.New(structType)
	vStruct := vPtr.Elem()
	for i, a := range args {
		if a == nil {
			continue
		}
		fieldValue, err := a.getValue(w, r)
		if err != nil {
			return reflect.Value{}, err
		}
		vStruct.Field(i).Set(fieldValue)
	}
	return vPtr, nil
}

func requestStruct(structType reflect.Type) (tagStructArgument, error) {
	if structType.Kind() != reflect.Struct {
		return tagStructArgument{}, errors.New("RequestStruct's argument must be a structure")
	}

	flags := flagArgument
	numFields := structType.NumField()
	numReadsBody := 0
	var arguments []Argument

	for i := 0; i < numFields; i++ {
		f := structType.Field(i)

		tag := f.Tag.Get(smartAPITagName)
		if len(tag) == 0 {
			arguments = append(arguments, nil)
			continue
		}

		fieldArg, err := parseArgument(tag, f.Type)
		if err != nil {
			return tagStructArgument{}, fmt.Errorf("(struct field %s) %w", f.Name, err)
		}

		if err := fieldArg.checkArg(f.Type); err != nil {
			return tagStructArgument{}, fmt.Errorf("(struct field %s) %w", f.Name, err)
		}

		fieldOpts := fieldArg.(EndpointParam).options()
		if fieldOpts.has(flagReadsRequestBody) {
			numReadsBody++
		}

		flags |= fieldOpts
		arguments = append(arguments, fieldArg)
	}

	if numReadsBody > 1 {
		return tagStructArgument{}, errors.New("only one struct field can read request's body")
	}

	return tagStructArgument{
		structType: structType,
		arguments:  arguments,
		flags:      flags,
	}, nil
}

// RequestStruct passes request's arguments into struct's fields by tags
func RequestStructDirect(s interface{}) EndpointParam {
	reqStruct, err := requestStruct(reflect.TypeOf(s))
	if err != nil {
		return errorEndpointParam{err: err}
	}
	return tagStructDirectArgument(reqStruct)
}

// RequestStruct passes request's arguments into struct's fields by tags
func RequestStruct(s interface{}) EndpointParam {
	reqStruct, err := requestStruct(reflect.TypeOf(s))
	if err != nil {
		return errorEndpointParam{err: err}
	}
	return reqStruct
}

type asIntArgument struct {
	arg Argument
}

func (a asIntArgument) options() endpointOptions {
	return a.arg.options()
}

func (a asIntArgument) checkArg(arg reflect.Type) error {
	if arg.Kind() != reflect.Int {
		return errors.New("argument must be an int")
	}
	return nil
}

func (a asIntArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	v, err := a.arg.getValue(w, r)
	if err != nil {
		return reflect.Value{}, err
	}

	intValue, err := strconv.Atoi(v.String())
	if err != nil {
		return reflect.Value{}, WrapError(http.StatusBadRequest, fmt.Errorf("AsInt(%s) conversion failed: %w", v.String(), err), "integer parse error")
	}

	return reflect.ValueOf(intValue), nil
}

func AsInt(param EndpointParam) EndpointParam {
	if !param.options().has(flagArgument) {
		return errorEndpointParam{err: errors.New("AsInt() requires an argument param")}
	}

	arg := param.(Argument)
	if err := arg.checkArg(reflect.TypeOf("")); err != nil {
		return errorEndpointParam{err: errors.New("argument must accept a string")}
	}

	return asIntArgument{arg: arg}
}

type asByteSliceArgument struct {
	arg Argument
}

func (a asByteSliceArgument) options() endpointOptions {
	return a.arg.options()
}

func (a asByteSliceArgument) checkArg(arg reflect.Type) error {
	if arg != byteSliceType {
		return errors.New("argument must be a byte slice")
	}
	return nil
}

func (a asByteSliceArgument) getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error) {
	v, err := a.arg.getValue(w, r)
	if err != nil {
		return reflect.Value{}, err
	}

	return reflect.ValueOf([]byte(v.String())), nil
}

func AsByteSlice(param EndpointParam) EndpointParam {
	if !param.options().has(flagArgument) {
		return errorEndpointParam{err: errors.New("AsByteSlice() requires an argument param")}
	}

	arg := param.(Argument)
	if err := arg.checkArg(reflect.TypeOf("")); err != nil {
		return errorEndpointParam{err: errors.New("argument must accept a string")}
	}

	return asByteSliceArgument{arg: arg}
}
