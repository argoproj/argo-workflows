package errors

type errTransient struct{ string }

func NewErrTransient(s string) error {
	return &errTransient{s}
}

func (e *errTransient) Is(target error) bool {
	_, ok := target.(*errTransient)
	return ok
}

func (e *errTransient) Error() string {
	return e.string
}
