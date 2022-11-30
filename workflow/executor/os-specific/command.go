package os_specific

import (
	"io"
	"os"
	"os/exec"

	"golang.org/x/term"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithField("argo", true)

func simpleStart(cmd *exec.Cmd) (func(), error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return func() {}, nil
}

func isTerminal(stdin io.Reader) bool {
	f, ok := stdin.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
