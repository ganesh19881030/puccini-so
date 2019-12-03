package components

type WfError struct {
	Message string
}

func (e *WfError) Error() string {
	return e.Message
}

func NewWfError(msg string) *WfError {
	err := &WfError{
		Message: msg,
	}

	return err
}

func (e *WfError) GetMessage() string {
	return e.Message
}
