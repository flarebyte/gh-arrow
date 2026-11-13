// Package diagnostic defines the stable error codes and process exit contract.
package diagnostic

import (
	"errors"
	"fmt"
	"io/fs"
)

type Error struct {
	Code, Message string
	Cause         error
}

func (e *Error) Error() string       { return e.Message }
func (e *Error) Unwrap() error       { return e.Cause }
func New(code, message string) error { return &Error{Code: code, Message: message} }
func Wrap(code, operation string, err error) error {
	return &Error{Code: code, Message: fmt.Sprintf("%s: %v", operation, err), Cause: err}
}
func Code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "ARROW_NOT_FOUND"
	case errors.Is(err, fs.ErrPermission):
		return "ARROW_PERMISSION"
	case errors.Is(err, fs.ErrExist):
		return "ARROW_EXISTS"
	default:
		return "ARROW_IO"
	}
}
func Exit(err error) int {
	if err == nil {
		return 0
	}
	switch Code(err) {
	case "ARROW_DRIFT":
		return 3
	case "ARROW_USAGE", "ARROW_CONFIG":
		return 2
	default:
		return 1
	}
}
