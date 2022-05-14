package os_specific

import "fmt"

// exitErr is a fake exec.ExecErr
type exitErr int

func (e exitErr) ExitCode() int {
	return int(e)
}

func (e exitErr) Error() string {
	return fmt.Sprintf("exit status %d", e)
}
