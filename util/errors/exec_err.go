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
