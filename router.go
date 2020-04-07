package smartapi

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
)

type Router interface {
	Use(middlewares ...func(http.Handler) http.Handler)
	With(middlewares ...func(http.Handler) http.Handler) Router
	AddEndpoint(method Method, pattern string, handler interface{}, args []EndpointParam)
	Post(pattern string, handler interface{}, args ...EndpointParam)
	Get(pattern string, handler interface{}, args ...EndpointParam)
	Put(pattern string, handler interface{}, args ...EndpointParam)
	Patch(pattern string, handler interface{}, args ...EndpointParam)
	Delete(pattern string, handler interface{}, args ...EndpointParam)
	Head(pattern string, handler interface{}, args ...EndpointParam)
	Options(pattern string, handler interface{}, args ...EndpointParam)
	Connect(pattern string, handler interface{}, args ...EndpointParam)
	Trace(pattern string, handler interface{}, args ...EndpointParam)
	Route(pattern string, handler RouteHandler, args ...EndpointParam)
}

type RouteHandler func(r Router)

type router struct {
	chiRouter chi.Router
	errors    []error
	logger    Logger
	params    []EndpointParam
}

var errType = reflect.TypeOf((*error)(nil)).Elem()
var byteType = reflect.TypeOf([]byte(nil))

func checkHandler(handlerFunc interface{}, arguments []Argument, writesResponse bool) (endpointHandler, error) {
	fnType := reflect.TypeOf(handlerFunc)
	if fnType.Kind() != reflect.Func {
		return nil, errors.New("handler must be a function")
	}

	if fnType.NumIn() != len(arguments) {
		return nil, errors.New("number of arguments of a function doesn't match provided arguments")
	}

	for i := 0; i < len(arguments); i++ {
		arg := fnType.In(i)
		if err := arguments[i].checkArg(arg); err != nil {
			return nil, fmt.Errorf("(argument %d) %w", i, err)
		}
	}

	switch fnType.NumOut() {
	case 0:
		return noResponseHandler{handlerFunc: handlerFunc}, nil
	case 1:
		outValue := fnType.Out(0)
		if !outValue.Implements(errType) {
			return nil, errors.New("expect an error type in return arguments")
		}
		return errorOnlyHandler{handlerFunc: handlerFunc}, nil
	case 2:
		if writesResponse {
			return nil, errors.New("cannot write response and return response")
		}

		errValue := fnType.Out(1)
		if !errValue.Implements(errType) {
			return nil, errors.New("expect an error type in return arguments")
		}

		value := fnType.Out(0)

		switch value.Kind() {
		case reflect.String:
			return stringErrorHandler{handlerFunc: handlerFunc}, nil
		case reflect.Slice:
			if value == byteType {
				return byteSliceErrorHandler{handlerFunc: handlerFunc}, nil
			}
			fallthrough
		case reflect.Ptr, reflect.Interface:
			return ptrErrorHandler{handlerFunc: handlerFunc}, nil
		case reflect.Struct:
			return structErrorHandler{handlerFunc: handlerFunc}, nil
		}

		return nil, errors.New("unsupported return type")
	}
	return nil, errors.New("invalid number of return arguments")
}

func isLegacyHandler(returnStatus int, args []Argument, handler interface{}) (http.HandlerFunc, bool) {
	switch len(args) {
	case 2:
		_, ok := args[0].(responseWriterArgument)
		if !ok {
			return nil, false
		}

		_, ok = args[1].(fullRequestArgument)
		if !ok {
			return nil, false
		}

		if returnStatus != 200 {
			return nil, false
		}
	case 0:
		if returnStatus != 0 {
			return nil, false
		}
	default:
		return nil, false
	}

	if h, ok := handler.(http.HandlerFunc); ok {
		return h, ok
	}

	h, ok := handler.(func(w http.ResponseWriter, h *http.Request))
	return h, ok
}

func (r *router) AddEndpoint(method Method, name string, handler interface{}, params []EndpointParam) {
	if handler == nil {
		r.errors = append(r.errors, fmt.Errorf("endpoint %s: nil handler", name))
		return
	}

	returnStatus := 0
	query := false
	writesResponse := false
	numReadsBody := 0

	joinedParams := append(r.params, params...)
	var args []Argument
	for i, a := range joinedParams {
		flags := a.options()
		if flags.has(flagArgument) {
			args = append(args, a.(Argument))
		}
		if flags.has(flagParsesQuery) {
			query = true
		}
		if flags.has(flagResponseStatus) {
			returnStatus = a.(responseStatusArgument).status
		}
		if flags.has(flagReadsRequestBody) {
			numReadsBody++
		}
		if flags.has(flagWritesResponse) {
			writesResponse = true
			if returnStatus == 0 {
				returnStatus = http.StatusOK
			}
		}
		if flags.has(flagError) {
			r.errors = append(r.errors, fmt.Errorf("endpoint %s: (argument %d) %w", name, i, a.(errorEndpointParam).err))
			return
		}
	}

	if h, ok := isLegacyHandler(returnStatus, args, handler); ok {
		r.chiRouter.MethodFunc(method.String(), name, h)
		return
	}

	if returnStatus == 0 {
		returnStatus = http.StatusNoContent
	}

	if numReadsBody > 1 {
		r.errors = append(r.errors, fmt.Errorf("endpoint %s: only one argument can read request's body", name))
	}

	endpointHandler, err := checkHandler(handler, args, writesResponse)
	if err != nil {
		r.errors = append(r.errors, fmt.Errorf("endpoint %s: %w", name, err))
	}

	if len(r.errors) > 0 {
		return
	}

	data := endpointData{
		arguments:    args,
		returnStatus: returnStatus,
		query:        query,
	}

	f := func(w http.ResponseWriter, rq *http.Request) {
		endpointHandler.handleRequest(w, rq, r.logger, data)
	}

	r.chiRouter.MethodFunc(method.String(), name, f)
}

// Use adds chi middlewares
func (r *router) Use(middlewares ...func(http.Handler) http.Handler) {
	r.chiRouter.Use(middlewares...)
}

// With returns a version of a handler with a middleware
func (r *router) With(middlewares ...func(http.Handler) http.Handler) Router {
	return &router{
		chiRouter: r.chiRouter.With(middlewares...),
		errors:    r.errors,
		logger:    r.logger,
	}
}

// Post adds an endpoint with a POST Method
func (r *router) Post(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodPost, pattern, handler, args)
}

// Get adds an endpoint with a GET Method
func (r *router) Get(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodGet, pattern, handler, args)
}

// Put adds an endpoint with a PUT Method
func (r *router) Put(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodPut, pattern, handler, args)
}

// Patch adds an endpoint with a PATCH Method
func (r *router) Patch(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodPatch, pattern, handler, args)
}

// Delete adds an endpoint with a DELETE Method
func (r *router) Delete(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodDelete, pattern, handler, args)
}

// Head adds an endpoint with a HEAD Method
func (r *router) Head(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodHead, pattern, handler, args)
}

// Options adds an endpoint with a OPTIONS Method
func (r *router) Options(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodOptions, pattern, handler, args)
}

// Connect adds an endpoint with a CONNECT Method
func (r *router) Connect(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodConnect, pattern, handler, args)
}

// Trace adds an endpoint with a TRACE Method
func (r *router) Trace(pattern string, handler interface{}, args ...EndpointParam) {
	r.AddEndpoint(MethodTrace, pattern, handler, args)
}

// Route routs endpoints to a specific path
func (r *router) Route(pattern string, handler RouteHandler, params ...EndpointParam) {
	if handler == nil {
		r.errors = append(r.errors, fmt.Errorf("route %s: nil handler", pattern))
		return
	}
	r.chiRouter.Route(pattern, func(rt chi.Router) {
		node := &router{
			logger:    r.logger,
			chiRouter: rt,
			params:    append(r.params, params...),
		}
		handler(node)
		for _, err := range node.errors {
			r.errors = append(r.errors, fmt.Errorf("route %s: %w", pattern, err))
		}
	})
}
