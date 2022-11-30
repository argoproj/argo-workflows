package os_specific

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

var logger = log.WithField("argo", true)

func StartCommand(cmd *exec.Cmd) (func(), error) {
	if isTerminal(cmd.Stdin) {
		logger.Warn("TTY detected but is not supported on windows")
	}
	return simpleStart(cmd)
}
