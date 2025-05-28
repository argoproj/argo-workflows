package osspecific

import (
	"os"
	"syscall"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

var (
	Term = syscall.SIGTERM
)

func CanIgnoreSignal(s os.Signal) bool {
	return s == syscall.SIGCHLD || s == syscall.SIGURG
}

func Kill(pid int, s syscall.Signal) error {
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		return syscall.Kill(-pgid, s)
	}
	return syscall.Kill(pid, s)
}

func Setpgid(a *syscall.SysProcAttr) {
	a.Setpgid = true
}

func Wait(process *os.Process) error {
	// We must copy the behaviour of Kubernetes in how we handle sub-processes.
	// Kubernetes only waits on PID 1, not on any sub-process that process might fork.
	// The only way for those forked processes to run in the background is to background the
	// sub-process by calling Process.Release.
	// Background processes always become zombies when they exit.
	// Because the sub-process is now running in the background it will become a zombie,
	// so we must wait for it.
	// Because we run the process in the background, we cannot Process.Wait for it to get the exit code.
	// Instead, we can reap it to get the exit code
	pid := process.Pid
	if err := process.Release(); err != nil {
		return err
	}

	for {
		var s syscall.WaitStatus
		wpid, err := syscall.Wait4(-1, &s, syscall.WNOHANG, nil)
		if err != nil {
			return err
		}
		if wpid == pid {
			if s.Exited() {
				return errors.NewExitErr(s.ExitStatus())
			} else if s.Signaled() {
				return errors.NewExitErr(128 + int(s.Signal()))
			}
		}
		time.Sleep(time.Second)
	}
}
