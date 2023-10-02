//go:build linux || darwin

package os_specific

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func StartCommand(cmd *exec.Cmd) (func(), error) {
	closer := func() {}

	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if !isTerminal(cmd.Stdin) {
		// avoid the error "Inappropriate ioctl for device" when
		// running in tty
		//
		// pty.Start uses setsid internally, which makes the process
		// the group leader already
		Setpgid(cmd.SysProcAttr)

		return simpleStart(cmd)
	}

	stdin, ok := cmd.Stdin.(*os.File)
	if !ok {
		// should never happen when stdin is a tty
		return nil, fmt.Errorf("Cannot convert stdin to os.File, it was %T", cmd.Stdin)
	}

	stdout := cmd.Stdout
	stderr := cmd.Stderr

	// pty.Start will not assign these to the pty unless they are nil
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// Handle pty size
	sigWinchCh := make(chan os.Signal, 1)
	signal.Notify(sigWinchCh, syscall.SIGWINCH)
	go func() {
		for range sigWinchCh {
			if err := pty.InheritSize(stdin, ptmx); err != nil {
				logger.WithError(err).Warn("Cannot resize pty")
			}
		}
	}()

	// Initial resize
	sigWinchCh <- syscall.SIGWINCH

	oldState, err := term.MakeRaw(int(stdin.Fd()))
	if err != nil {
		return nil, err
	}

	go func() { _, _ = io.Copy(ptmx, stdin) }()
	go func() { _, _ = io.Copy(stdout, ptmx) }()
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
