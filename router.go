package smartapi

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
)

type Router interface {
	Post(pattern string, handler interface{}, args ...EndpointParam)
	Get(pattern string, handler interface{}, args ...EndpointParam)
}

type route struct {
	pattern string
	node    *routeNode
}

type RouteHandler func(r Router)

type routeNode struct {
	errors      []error
	endpoints   []endpoint
	logger      Logger
	middlewares []func(http.Handler) http.Handler
	subRoutes   []route
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

func (r *routeNode) addEndpoint(method method, name string, handler interface{}, args []EndpointParam) {
	returnStatus := 0
	query := false
	writesResponse := false
	numReadsBody := 0

	var params []Argument
	var middlewares []func(http.Handler) http.Handler
	for i, a := range args {
		flags := a.options()
		if flags.has(flagArgument) {
			params = append(params, a.(Argument))
		}
		if flags.has(flagParsesQuery) {
			query = true
		}
		if flags.has(flagResponseStatus) {
			returnStatus = a.(responseStatusArgument).status
		}
		if flags.has(flagMiddleware) {
			middlewares = append(middlewares, a.(middleware).middlewares...)
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

	if h, ok := isLegacyHandler(returnStatus, params, handler); ok {
		r.endpoints = append(r.endpoints, endpoint{
			name:         name,
			method:       method,
			arguments:    nil,
			handler:      legacyHandler{handlerFunc: h},
			returnStatus: 0,
			query:        false,
			cookies:      false,
			legacy:       true,
			middlewares:  middlewares,
		})
		return
	}

	if returnStatus == 0 {
		returnStatus = http.StatusNoContent
	}

	if numReadsBody > 1 {
		r.errors = append(r.errors, fmt.Errorf("endpoint %s: only one argument can read request's body", name))
	}

	endpointHandler, err := checkHandler(handler, params, writesResponse)
	if err != nil {
		r.errors = append(r.errors, fmt.Errorf("endpoint %s: %w", name, err))
	}

	if len(r.errors) > 0 {
		return
	}

	r.endpoints = append(r.endpoints, endpoint{
		name:         name,
		arguments:    params,
		handler:      endpointHandler,
		method:       method,
		returnStatus: returnStatus,
		query:        query,
		middlewares:  middlewares,
		legacy:       false,
	})
}

// With adds chi middlewares
func (r *routeNode) Use(middlewares ...func(http.Handler) http.Handler) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// Post adds an endpoint with a POST method
func (r *routeNode) Post(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodPost, pattern, handler, args)
}

// Get adds an endpoint with a GET method
func (r *routeNode) Get(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodGet, pattern, handler, args)
}

// Put adds an endpoint with a PUT method
func (r *routeNode) Put(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodPut, pattern, handler, args)
}

// Patch adds an endpoint with a PATCH method
func (r *routeNode) Patch(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodPatch, pattern, handler, args)
}

// Delete adds an endpoint with a DELETE method
func (r *routeNode) Delete(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodDelete, pattern, handler, args)
}

// Head adds an endpoint with a HEAD method
func (r *routeNode) Head(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodHead, pattern, handler, args)
}

// Options adds an endpoint with a OPTIONS method
func (r *routeNode) Options(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodOptions, pattern, handler, args)
}

// Connect adds an endpoint with a CONNECT method
func (r *routeNode) Connect(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodConnect, pattern, handler, args)
}

// Trace adds an endpoint with a TRACE method
func (r *routeNode) Trace(pattern string, handler interface{}, args ...EndpointParam) {
	r.addEndpoint(methodTrace, pattern, handler, args)
}

// Handler returns an http.Handler of the API
func (r *routeNode) chiRouter(router chi.Router) error {
	if len(r.errors) != 0 {
		errMsg := r.errors[0].Error()
		for _, e := range r.errors[1:] {
			errMsg += ", " + e.Error()
		}
		return errors.New(errMsg)
	}

	if len(r.middlewares) > 0 {
		router.Use(r.middlewares...)
	}

	for _, subRoute := range r.subRoutes {
		var err error
		router.Route(subRoute.pattern, func(router chi.Router) {
			err = subRoute.node.chiRouter(router)
		})
		if err != nil {
			return err
		}
	}

	for _, e := range r.endpoints {
		var f http.HandlerFunc
		if e.legacy {
			f = e.handler.(legacyHandler).handlerFunc
		} else {
			f = func(e endpoint) http.HandlerFunc {
				return func(w http.ResponseWriter, rq *http.Request) {
					e.handler.handleRequest(w, rq, r.logger, e)
				}
			}(e)
		}
		var subRouter chi.Router
		if len(e.middlewares) > 0 {
			subRouter = router.With(e.middlewares...)
		} else {
			subRouter = router
		}
		subRouter.MethodFunc(e.method.String(), e.name, f)
	}

	return nil
}

func (r *routeNode) Route(pattern string, rh RouteHandler, args ...EndpointParam) {
	node := &routeNode{
		logger: r.logger,
	}
	if rh != nil {
		rh(node)
	}
	r.subRoutes = append(r.subRoutes, route{
		pattern: pattern,
		node:    node,
	})
}
