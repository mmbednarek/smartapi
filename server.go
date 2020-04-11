package smartapi

import (
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
