package errs

import (
	"fmt"
	"net/http"
	"strings"
)

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

func (f FieldError) String() string {
	return fmt.Sprintf("%s: %s", f.Field, f.Message)
}

type HTTPError struct {
	Status  int          `json:"-"`
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

func (e *HTTPError) Error() string {
	if len(e.Fields) == 0 {
		return fmt.Sprintf("%d %s: %s", e.Status, e.Code, e.Message)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d %s: %s", e.Status, e.Code, e.Message)
	for _, f := range e.Fields {
		fmt.Fprintf(&sb, "\n  %s", f.String())
	}
	return sb.String()
}

func New(status int, code, message string) *HTTPError {
	return &HTTPError{Status: status, Code: code, Message: message}
}

func WithFields(status int, code, message string, fields []FieldError) *HTTPError {
	return &HTTPError{Status: status, Code: code, Message: message, Fields: fields}
}

func BadRequest(code, message string) *HTTPError {
	return New(http.StatusBadRequest, code, message)
}

func Invalid(message string, fields []FieldError) *HTTPError {
	return WithFields(http.StatusBadRequest, "invalid", message, fields)
}

func Unauthorized(message string) *HTTPError {
	return New(http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(message string) *HTTPError {
	return New(http.StatusForbidden, "forbidden", message)
}

func NotFound(code, message string) *HTTPError {
	return New(http.StatusNotFound, code, message)
}

func Conflict(code, message string) *HTTPError {
	return New(http.StatusConflict, code, message)
}

func Internal(message string) *HTTPError {
	return New(http.StatusInternalServerError, "internal", message)
}
