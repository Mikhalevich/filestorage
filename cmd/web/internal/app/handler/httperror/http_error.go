package httperror

import (
	"fmt"
	"net/http"
)

type HTTPError struct {
	Status int
	Msg    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("status: %d, msg: %s", e.Status, e.Msg)
}

func InternalServerError(msg string) *HTTPError {
	return &HTTPError{
		Status: http.StatusInternalServerError,
		Msg:    msg,
	}
}

func BadRequest(msg string) *HTTPError {
	return &HTTPError{
		Status: http.StatusBadRequest,
		Msg:    msg,
	}
}

func NotFound(msg string) *HTTPError {
	return &HTTPError{
		Status: http.StatusNotFound,
		Msg:    msg,
	}
}

func Conflict(msg string) *HTTPError {
	return &HTTPError{
		Status: http.StatusConflict,
		Msg:    msg,
	}
}
