//go:build windows
package os_specific

import (
	"os"
	"os/exec"
	"io"
)

func StartCommand(cmd *exec.Cmd, stdin *os.File, stdout io.Writer, stderr io.Writer) (func(), error) {
	closer := func() {}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return closer, nil
}
