package smartapi

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
)

type endpoint struct {
	name         string
	method       method
	arguments    []Argument
	handler      endpointHandler
	returnStatus int
	query        bool
	cookies      bool
	middlewares  []func(http.Handler) http.Handler
}

// Server handles http endpoints
type Server struct {
	errors      []error
	endpoints   []endpoint
	logger      Logger
	middlewares []func(http.Handler) http.Handler
}

// StartAPI starts a user defined API
func StartAPI(a API, address string) error {
	a.Init()
	if err := a.Start(address); err != nil {
		return err
	}
	return nil
}

// NewServer constructs a server
func NewServer(logger Logger) *Server {
	return &Server{logger: logger}
}

var errType = reflect.TypeOf((*error)(nil)).Elem()
var byteType = reflect.TypeOf([]byte(nil))

func checkHandler(handlerFunc interface{}, arguments []Argument) (endpointHandler, error) {
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
			return nil, err
		}
	}

	switch fnType.NumOut() {
	case 1:
		outValue := fnType.Out(0)
		if !outValue.Implements(errType) {
			return nil, errors.New("expect an error type in return arguments")
		}
		return errorOnlyHandler{handlerFunc: handlerFunc}, nil
	case 2:
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

func (s *Server) addEndpoint(method method, name string, handler interface{}, args []EndpointParam) {
	returnStatus := http.StatusNoContent
	query := false

	var params []Argument
	var middlewares []func(http.Handler) http.Handler
	for _, a := range args {
		switch a.(type) {
		case responseStatusArgument:
			returnStatus = a.(responseStatusArgument).status
		case middlewareArgument:
			middlewares = append(middlewares, a.(middlewareArgument).middlewares...)
		case queryParamArgument:
			query = true
			params = append(params, a.(Argument))
		case Argument:
			params = append(params, a.(Argument))
		}
	}

	endpointHandler, err := checkHandler(handler, params)
	if err != nil {
		s.errors = append(s.errors, fmt.Errorf("endpoint %s: %w", name, err))
		return
	}
	s.endpoints = append(s.endpoints, endpoint{
		name:         name,
		arguments:    params,
		handler:      endpointHandler,
		method:       method,
		returnStatus: returnStatus,
		query:        query,
		middlewares:  middlewares,
	})
}

// With adds chi middlewares
func (s *Server) With(middlewares ...func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middlewares...)
}

// Post adds an endpoint with a POST method
func (s *Server) Post(pattern string, handler interface{}, args ...EndpointParam) {
	s.addEndpoint(methodPost, pattern, handler, args)
}

// Get adds an endpoint with a GET method
func (s *Server) Get(pattern string, handler interface{}, args ...EndpointParam) {
	s.addEndpoint(methodGet, pattern, handler, args)
}

// Put adds an endpoint with a PUT method
func (s *Server) Put(pattern string, handler interface{}, args ...EndpointParam) {
	s.addEndpoint(methodPut, pattern, handler, args)
}

// Patch adds an endpoint with a PATCH method
func (s *Server) Patch(pattern string, handler interface{}, args ...EndpointParam) {
	s.addEndpoint(methodPatch, pattern, handler, args)
}

// Delete adds an endpoint with a DELETE method
func (s *Server) Delete(pattern string, handler interface{}, args ...EndpointParam) {
	s.addEndpoint(methodDelete, pattern, handler, args)
}

// Handler returns an http.Handler of the API
func (s *Server) Handler() (http.Handler, error) {
	r := chi.NewRouter()

	if len(s.errors) != 0 {
		errMsg := s.errors[0].Error()
		for _, e := range s.errors[1:] {
			errMsg += ", " + e.Error()
		}
		return nil, errors.New(errMsg)
	}

	var router chi.Router
	if len(s.middlewares) > 0 {
		router = r.With(s.middlewares...)
	} else {
		router = r
	}

	for _, e := range s.endpoints {
		f := func(e endpoint) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				e.handler.handleRequest(w, r, s.logger, e)
			}
		}(e)

		if len(e.middlewares) > 0 {
			router = router.With(e.middlewares...)
		}

		switch e.method {
		case methodPost:
			router.Post(e.name, f)
		case methodGet:
			router.Get(e.name, f)
		case methodPatch:
			router.Patch(e.name, f)
		case methodPut:
			router.Put(e.name, f)
		case methodDelete:
			router.Delete(e.name, f)
		}
	}

	return router, nil
}

// Start starts the api
func (s *Server) Start(address string) error {
	handler, err := s.Handler()
	if err != nil {
		return err
	}
	if err := http.ListenAndServe(address, handler); err != nil {
		return fmt.Errorf("ListenAndServe: %w", err)
	}
	return nil
}
