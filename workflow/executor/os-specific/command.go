package os_specific

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

var logger = log.WithField("argo", true)

func isTerminal(stdin io.Reader) bool {
	f, ok := stdin.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
