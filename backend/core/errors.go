package core

import "github.com/pkg/errors"

// FieldError is used to indicate an error with a specific struct field.
type FieldError struct {
	Field string
	Error string
}

type ValidationError struct {
	Err    error
	Fields []FieldError
}

func NewValidationError(err error, flds ...FieldError) error {
	return &ValidationError{err, flds}
}

func (err ValidationError) Error() string {
	if err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

type shutdown struct {
	Message string
}

func NewShutdownError(msg string) error {
	return &shutdown{Message: msg}
}

func (s shutdown) Error() string {
	return s.Message
}

func IsShutdown(err error) bool {
	if _, ok := errors.Cause(err).(*shutdown); ok {
		return true
	}
	return false
}
