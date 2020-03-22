package smartapi

import "fmt"

func Error(status int, logMsg string, reason string) ApiError {
	return statusError{
		errCode: status,
		message: logMsg,
		reason:  reason,
	}
}

func Errorf(status int, format string, args ...interface{}) ApiError {
	return statusError{
		errCode: status,
		message: fmt.Sprintf(format, args...),
		reason:  "unknown",
	}
}

func WrapError(status int, err error, reason string) ApiError {
	return statusError{
		errCode: status,
		message: err.Error(),
		reason:  reason,
	}
}

type errorResponse struct {
	Status int    `json:"status"`
	Reason string `json:"reason"`
}

type statusError struct {
	errCode int
	message string
	reason  string
}

func (s statusError) Reason() string {
	return s.reason
}

func (s statusError) Status() int {
	return s.errCode
}

func (s statusError) Error() string {
	return s.message
}
