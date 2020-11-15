package apps

type ArgumentError struct {
	msg string
}

func NewArgumentError(msg string) *ArgumentError {
	return &ArgumentError{msg}
}

func (err *ArgumentError) Error() string {
	return err.msg
}
