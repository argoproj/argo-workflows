package osspecific

import (
	"io"
	"os"

	"golang.org/x/term"
)

func isTerminal(stdin io.Reader) bool {
	f, ok := stdin.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
