package smartapi

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
)

type Endpoint struct {
	name         string
	method       method
	arguments    []Argument
	handler      endpointHandler
	returnStatus int
	query        bool
	cookies      bool
	middlewares  []func(http.Handler) http.Handler
}

type Server struct {
	errors      []error
	endpoints   []Endpoint
	logger      Logger
	middlewares []func(http.Handler) http.Handler
}

func StartAPI(a API, address string) error {
	a.Init()
	if err := a.Start(address); err != nil {
		return err
	}
	return nil
}

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
			} else {
				fmt.Printf("%v != %v", value.Elem(), byteType)
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

func (s *Server) addEndpoint(method method, name string, handler interface{}, args []Argument) {
	returnStatus := http.StatusNoContent
	query := false

	var params []Argument
	var middlewares []func(http.Handler) http.Handler
	for _, a := range args {
		switch a.(type) {
		case returnStatusArgument:
			returnStatus = a.(returnStatusArgument).status
		case middlewareArgument:
			middlewares = append(middlewares, a.(middlewareArgument).middlewares...)
		case queryParamArgument:
			query = true
			params = append(params, a)
		default:
			params = append(params, a)
		}
	}

	endpointHandler, err := checkHandler(handler, params)
	if err != nil {
		s.errors = append(s.errors, fmt.Errorf("endpoint %s: %w", name, err))
		return
	}
	s.endpoints = append(s.endpoints, Endpoint{
		name:         name,
		arguments:    params,
		handler:      endpointHandler,
		method:       method,
		returnStatus: returnStatus,
		query:        query,
		middlewares:  middlewares,
	})
}

func (s *Server) With(middlewares ...func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middlewares...)
}

func (s *Server) Post(pattern string, handler interface{}, args ...Argument) {
	s.addEndpoint(POST, pattern, handler, args)
}

func (s *Server) Get(pattern string, handler interface{}, args ...Argument) {
	s.addEndpoint(GET, pattern, handler, args)
}

func (s *Server) Put(pattern string, handler interface{}, args ...Argument) {
	s.addEndpoint(PUT, pattern, handler, args)
}

func (s *Server) Patch(pattern string, handler interface{}, args ...Argument) {
	s.addEndpoint(PATCH, pattern, handler, args)
}

func (s *Server) Delete(pattern string, handler interface{}, args ...Argument) {
	s.addEndpoint(DELETE, pattern, handler, args)
}

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
	router = r.With(s.middlewares...)

	for _, e := range s.endpoints {
		f := func(e Endpoint) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				e.handler.HandleRequest(w, r, s.logger, e)
			}
		}(e)

		var epRouter chi.Router
		if len(e.middlewares) > 0 {
			epRouter = router.With(e.middlewares...)
		} else {
			epRouter = router
		}

		switch e.method {
		case POST:
			epRouter.Post(e.name, f)
		case GET:
			epRouter.Get(e.name, f)
		case PATCH:
			epRouter.Patch(e.name, f)
		case PUT:
			epRouter.Put(e.name, f)
		case DELETE:
			epRouter.Delete(e.name, f)
		}
	}

	return router, nil
}

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
