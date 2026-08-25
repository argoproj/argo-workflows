package errors

import "fmt"

type Exited interface {
	ExitCode() int
}

func NewExitErr(exitCode int) error {
	if exitCode > 0 {
		return execErr(exitCode)
	}
	return nil
}

type execErr int

func (e execErr) ExitCode() int {
	return int(e)
}

func (e execErr) Error() string {
	return fmt.Sprintf("exit status %d", e)
}

// exitErrWithCause pairs an explicit exit code with an underlying error. It lets
// a caller both control the process exit code (via Exited) and preserve a
// descriptive cause. Unlike a bare exit-status error, it wraps a cause, which is
// how main() decides the message is worth writing to the termination log.
type exitErrWithCause struct {
	code  int
	cause error
}

func (e exitErrWithCause) ExitCode() int { return e.code }
func (e exitErrWithCause) Error() string { return e.cause.Error() }
func (e exitErrWithCause) Unwrap() error { return e.cause }

// NewExitErrWithCause returns an Exited error reporting exitCode while preserving
// cause's message and unwrap chain. Use it when the process must exit with a
// specific code yet still surface why; main() writes the cause as the container's
// termination message.
func NewExitErrWithCause(exitCode int, cause error) error {
	return exitErrWithCause{code: exitCode, cause: cause}
}
