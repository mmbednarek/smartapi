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

type method int

const (
	methodPost method = iota
	methodGet
	methodPatch
	methodDelete
	methodPut
	methodOptions
	methodConnect
	methodHead
	methodTrace
)
func (m method) String() string {
	return methodNames[m]
}

var methodNames = []string{
	methodPost:    "POST",
	methodGet:     "GET",
	methodPatch:   "PATCH",
	methodDelete:  "DELETE",
	methodPut:     "PUT",
	methodOptions: "OPTIONS",
	methodConnect: "CONNECT",
	methodHead:    "HEAD",
	methodTrace:   "TRACE",
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
