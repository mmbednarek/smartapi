package smartapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

type endpointData struct {
	arguments    []Argument
	returnStatus int
	query        bool
}

// Server handles http endpoints
type Server struct {
	router
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
		router: router{
			chiRouter: chi.NewRouter(),
			logger:    logger,
		},
	}
}

// Handler returns an http.Handler of the API
func (s *Server) Handler() (http.Handler, error) {
	if len(s.errors) != 0 {
		errMsg := s.errors[0].Error()
		for _, e := range s.errors[1:] {
			errMsg += ", " + e.Error()
		}
		return nil, errors.New(errMsg)
	}
	return s.router.chiRouter, nil
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
