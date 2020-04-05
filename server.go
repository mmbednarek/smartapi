package smartapi

import (
	"fmt"
	"net/http"

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
	legacy       bool
	middlewares  []func(http.Handler) http.Handler
}

// Server handles http endpoints
type Server struct {
	routeNode
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
	return &Server{
		routeNode: routeNode{
			logger: logger,
		},
	}
}

// Handler returns an http.Handler of the API
func (s *Server) Handler() (http.Handler, error) {
	r := chi.NewRouter()
	if err := s.chiRouter(r); err != nil {
		return nil, err
	}
	return r, nil
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
