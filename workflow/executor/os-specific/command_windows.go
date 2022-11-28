package os_specific

import (
	"io"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

var logger = log.WithField("argo", true)

func StartCommand(cmd *exec.Cmd, stdin *os.File, stdout io.Writer, stderr io.Writer) (func(), error) {
	if term.IsTerminal(int(stdin.Fd())) {
		logger.Warn("TTY detected but is not supported on windows")
	}
	return simpleStart(cmd, stdout, stderr)
}
