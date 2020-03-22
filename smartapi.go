package smartapi

import (
	"context"
	"log"
	"net/http"
	"reflect"
)

type endpointHandler interface {
	HandleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint Endpoint)
}

type Argument interface {
	checkArg(arg reflect.Type) error
	getValue(w http.ResponseWriter, r *http.Request) (reflect.Value, error)
}

type ApiError interface {
	error
	Status() int
	Reason() string
}

type Logger interface {
	LogApiError(ctx context.Context, err ApiError)
	LogError(ctx context.Context, err error)
}

type API interface {
	Start(string) error
	Init()
	Handler() (http.Handler, error)
}

type method int

const (
	POST method = iota
	GET
	PATCH
	DELETE
	PUT
)

type defaultLogger struct{}

func (l defaultLogger) LogApiError(ctx context.Context, err ApiError) {
	log.Printf("[%d] %s", err.Status(), err.Error())
}

func (l defaultLogger) LogError(ctx context.Context, err error) {
	log.Print(err)
}

var DefaultLogger Logger = defaultLogger{}
