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

// Method represents an http method
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

// String converts a method into a string with name of the method in capital letters
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

func (defaultLogger) LogApiError(ctx context.Context, err ApiError) {
	log.Printf("[%d] %s", err.Status(), err.Error())
}

func (defaultLogger) LogError(ctx context.Context, err error) {
	log.Print(err)
}

// DefaultLogger is simple implementation of the Logger interface
var DefaultLogger Logger = defaultLogger{}
