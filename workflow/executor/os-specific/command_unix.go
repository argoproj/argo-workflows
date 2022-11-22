//go:build !windows
package os_specific


import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func StartCommand(cmd *exec.Cmd, stdin *os.File, stdout io.Writer, stderr io.Writer) (func(), error) {

	closer := func() {}

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if !term.IsTerminal(int(stdin.Fd())) {
		// avoid the error "Inappropriate ioctl for device" when
		// running in tty
		//
		// pty.Start uses setsid internally, which makes the process
		// the group leader already
		Setpgid(cmd.SysProcAttr)

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Start(); err != nil {
			return nil, err
		}

		return closer, nil
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// Handle pty size
	sigWinchCh := make(chan os.Signal, 1)
	signal.Notify(sigWinchCh, syscall.SIGWINCH)
	go func() {
		for range sigWinchCh {
			// TODO: log error somehow?
			_ = pty.InheritSize(stdin, ptmx)
		}
	}()

	// Initial resize
	sigWinchCh <- syscall.SIGWINCH

	// Set stdin in raw mode
	oldState, err := term.MakeRaw(int(stdin.Fd()))
	if err != nil {
		return nil, err
	}

	// copy from stdin to the pty
	go func() { _, _ = io.Copy(ptmx, stdin) }()
	// copy from pty to stdout
	go func() { _, _ = io.Copy(stdout, ptmx) }()
	// copy from pty to stderr
	go func() { _, _ = io.Copy(stderr, ptmx) }()

	origCloser := closer
	closer = func() {
		signal.Stop(sigWinchCh)
		close(sigWinchCh)

		_ = term.Restore(int(stdin.Fd()), oldState)
		_ = ptmx.Close()
		origCloser()
	}

	return closer, nil
}
