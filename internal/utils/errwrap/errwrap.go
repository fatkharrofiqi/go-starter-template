package errwrap

import (
	"errors"
)

// Custom error types
var (
	ErrNotFound     = errors.New("resource not found")
	ErrValidation   = errors.New("validation failed")
	ErrUnauthorized = errors.New("unauthorized access")
	ErrInternal     = errors.New("internal server error")
	ErrBadRequest   = errors.New("bad request")
	ErrDataExists   = errors.New("data already exists")
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// customError wraps an error with a custom message
type customError struct {
	msg string
	err error
}

// Error returns only the custom message
func (e *customError) Error() string {
	return e.msg
}

// Unwrap returns the underlying error for errors.Is
func (e *customError) Unwrap() error {
	return e.err
}

// WrapError creates a wrapped error with only the custom message
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	if message == "" {
		return err
	}
	return &customError{
		msg: message,
		err: err,
	}
}

// IsErrorType checks if an error matches a specific error type
func IsErrorType(err error, target error) bool {
	return errors.Is(err, target)
}
