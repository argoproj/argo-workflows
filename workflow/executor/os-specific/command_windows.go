package os_specific

import (
	"os"
	"os/exec"
)

func StartCommand(cmd *exec.Cmd) (func(), error) {
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}

	if isTerminal(cmd.Stdin) {
		logger.Warn("TTY detected but is not supported on windows")
	}
	return simpleStart(cmd)
}
