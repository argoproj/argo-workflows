//go:build linux || darwin

package osspecific

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func StartCommand(ctx context.Context, cmd *exec.Cmd) (func(), error) {
	logger := logging.RequireLoggerFromContext(ctx).WithField("argo", true)
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
		return nil, fmt.Errorf("cannot convert stdin to os.File, it was %T", cmd.Stdin)
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
				logger.WithError(err).Warn(ctx, "Cannot resize pty")
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

func simpleStart(cmd *exec.Cmd) (func(), error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	closer := func() {
		cmd.WaitDelay = 100 * time.Millisecond
		_ = cmd.Wait()
	}

	return closer, nil
}
