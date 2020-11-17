package utils

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

func (err *ValidationError) Error() string {
	if err.Err == nil {
		return ""
	}
	return err.Err.Error()
}
