package smartapi

import (
	"context"
	"log"
	"net/http"
	"reflect"
)

type endpointHandler interface {
	HandleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpoint)
}

// Argument is used with endpoint definition
type Argument interface {
	checkArg(arg reflect.Type) error
	getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error)
}

// ApiError represents an API error
type ApiError interface {
	error
	Status() int
	Reason() string
}

// Logger logs the outcome of unsuccessful http requests
type Logger interface {
	LogApiError(ctx context.Context, err ApiError)
	LogError(ctx context.Context, err error)
}

// API interface represents an API
type API interface {
	Start(string) error
	Init()
	Handler() (http.Handler, error)
}

type method int

const (
	methodPost method = iota
	methodGet
	methodPatch
	methodDelete
	methodPut
)

type defaultLogger struct{}

func (l defaultLogger) LogApiError(ctx context.Context, err ApiError) {
	log.Printf("[%d] %s", err.Status(), err.Error())
}

func (l defaultLogger) LogError(ctx context.Context, err error) {
	log.Print(err)
}

// DefaultLogger is simple implementation of the Logger interface
var DefaultLogger Logger = defaultLogger{}
