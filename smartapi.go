package smartapi

import (
	"context"
	"log"
)

// Logger logs the outcome of unsuccessful http requests
type Logger interface {
	LogApiError(ctx context.Context, err ApiError)
	LogError(ctx context.Context, err error)
}

// API interface represents an API
type API interface {
	Start(string) error
	Init()
}

type Method int

const (
	MethodPost Method = iota
	MethodGet
	MethodPatch
	MethodDelete
	MethodPut
	MethodOptions
	MethodConnect
	MethodHead
	MethodTrace
)
func (m Method) String() string {
	return methodNames[m]
}

var methodNames = []string{
	MethodPost:    "POST",
	MethodGet:     "GET",
	MethodPatch:   "PATCH",
	MethodDelete:  "DELETE",
	MethodPut:     "PUT",
	MethodOptions: "OPTIONS",
	MethodConnect: "CONNECT",
	MethodHead:    "HEAD",
	MethodTrace:   "TRACE",
}

type defaultLogger struct{}

func (l defaultLogger) LogApiError(ctx context.Context, err ApiError) {
	log.Printf("[%d] %s", err.Status(), err.Error())
}

func (l defaultLogger) LogError(ctx context.Context, err error) {
	log.Print(err)
}

// DefaultLogger is simple implementation of the Logger interface
var DefaultLogger Logger = defaultLogger{}
