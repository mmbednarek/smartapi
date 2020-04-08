package smartapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
)

type endpointHandler interface {
	handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData)
}

func getCallAttributes(w http.ResponseWriter, r *http.Request, endpoint endpointData) ([]reflect.Value, error) {
	if endpoint.query {
		if err := r.ParseForm(); err != nil {
			return nil, WrapError(http.StatusBadRequest, err, "could not parse form")
		}
	}

	var result []reflect.Value
	for _, a := range endpoint.arguments {
		value, err := a.getValue(w, r)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func handleErrorValue(ctx context.Context, w http.ResponseWriter, logger Logger, errorValue reflect.Value) {
	err, ok := errorValue.Interface().(error)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(errorResponse{
			Status: http.StatusInternalServerError,
			Reason: "invalid API construction",
		})
		return
	}
	handleError(ctx, w, logger, err)
}

func handleError(ctx context.Context, w http.ResponseWriter, logger Logger, err error) {
	var apiErr ApiError
	if errors.As(err, &apiErr) {
		if logger != nil {
			logger.LogApiError(ctx, apiErr)
		}
	} else {
		if logger != nil {
			logger.LogError(ctx, err)
		}
		apiErr = statusError{
			errCode: http.StatusInternalServerError,
			message: err.Error(),
			reason:  "unknown",
		}
	}

	w.WriteHeader(apiErr.Status())
	_ = json.NewEncoder(w).Encode(errorResponse{
		Status: apiErr.Status(),
		Reason: apiErr.Reason(),
	})
}

type noResponseHandler struct {
	handlerFunc interface{}
}

func (e noResponseHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(e.handlerFunc)
	value.Call(attribs)
	w.WriteHeader(endpoint.returnStatus)
}

type errorOnlyHandler struct {
	handlerFunc interface{}
}

func (e errorOnlyHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(e.handlerFunc)
	result := value.Call(attribs)

	errorValue := result[0]

	if !errorValue.IsNil() {
		handleErrorValue(r.Context(), w, logger, errorValue)
		return
	}

	w.WriteHeader(endpoint.returnStatus)
}

type ptrErrorHandler struct {
	handlerFunc interface{}
}

func (e ptrErrorHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(e.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0]
	errorValue := result[1]

	if !errorValue.IsNil() {
		handleErrorValue(r.Context(), w, logger, errorValue)
		return
	}

	if responseValue.IsNil() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(responseValue.Interface()); err != nil {
		handleError(r.Context(), w, logger, WrapError(http.StatusInternalServerError, err, "cannot encode response"))
		return
	}
}

type ptrHandler struct {
	handlerFunc interface{}
}

func (e ptrHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(e.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0]

	if responseValue.IsNil() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(responseValue.Interface()); err != nil {
		handleError(r.Context(), w, logger, WrapError(http.StatusInternalServerError, err, "cannot encode response"))
		return
	}
}

type structErrorHandler struct {
	handlerFunc interface{}
}

func (s structErrorHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(s.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0]
	errorValue := result[1]

	if !errorValue.IsNil() {
		handleErrorValue(r.Context(), w, logger, errorValue)
		return
	}

	if err := json.NewEncoder(w).Encode(responseValue.Interface()); err != nil {
		handleError(r.Context(), w, logger, WrapError(http.StatusInternalServerError, err, "cannot encode response"))
		return
	}
}

type structHandler struct {
	handlerFunc interface{}
}

func (s structHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(s.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0]

	if err := json.NewEncoder(w).Encode(responseValue.Interface()); err != nil {
		handleError(r.Context(), w, logger, WrapError(http.StatusInternalServerError, err, "cannot encode response"))
		return
	}
}

type stringErrorHandler struct {
	handlerFunc interface{}
}

func (s stringErrorHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(s.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0].String()
	errorValue := result[1]

	if !errorValue.IsNil() {
		handleErrorValue(r.Context(), w, logger, errorValue)
		return
	}

	if len(responseValue) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = w.Write([]byte(responseValue))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type stringHandler struct {
	handlerFunc interface{}
}

func (s stringHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(s.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0].String()

	if len(responseValue) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = w.Write([]byte(responseValue))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type byteSliceErrorHandler struct {
	handlerFunc interface{}
}

func (b byteSliceErrorHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(b.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0].Bytes()
	errorValue := result[1]

	if !errorValue.IsNil() {
		handleErrorValue(r.Context(), w, logger, errorValue)
		return
	}

	if len(responseValue) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = w.Write(responseValue)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type byteSliceHandler struct {
	handlerFunc interface{}
}

func (b byteSliceHandler) handleRequest(w http.ResponseWriter, r *http.Request, logger Logger, endpoint endpointData) {
	attribs, err := getCallAttributes(w, r, endpoint)
	if err != nil {
		handleError(r.Context(), w, logger, err)
		return
	}
	value := reflect.ValueOf(b.handlerFunc)
	result := value.Call(attribs)

	responseValue := result[0].Bytes()

	if len(responseValue) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = w.Write(responseValue)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
