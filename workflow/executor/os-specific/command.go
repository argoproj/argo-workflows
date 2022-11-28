package os_specific

import (
	"io"
	"os/exec"
)

func simpleStart(cmd *exec.Cmd, stdout io.Writer, stderr io.Writer) (func(), error) {
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return func() {}, nil
}
