package cloudsigma

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyPayloadNotAllowed = errors.New("empty payload not allowed")
	ErrEmptyArgument          = errors.New("argument cannot be empty")
)

// An ErrorResponse reports one or more errors caused by an API request.
//
// CloudSigma API docs: http://cloudsigma-docs.readthedocs.io/en/latest/errors.html
type ErrorResponse struct {
	Response *Response // HTTP response that caused this error.
	Errors   []Error
}

type Error struct {
	Message string `json:"error_message"`
	Point   string `json:"error_point"`
	Type    string `json:"error_type"`
}

func (r *ErrorResponse) Error() string {
	if r.Response.RequestID != "" {
		return fmt.Sprintf("%v %v: %d (request %q) %+v",
			r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Response.RequestID, r.Errors)
	}
	return fmt.Sprintf("%v %v: %d %+v",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Errors)
}
