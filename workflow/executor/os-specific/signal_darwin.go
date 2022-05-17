package os_specific

import (
	"os"
	"syscall"
	"time"

	"github.com/mitchellh/go-ps"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

func IsSIGCHLD(s os.Signal) bool { return s == syscall.SIGCHLD }

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
	// Kubernetes only waits on PID 1, not on forked process that process might fork.
	// The only way for those forked processes to run in the background is to background the
	// sub-process by calling Process.Release.
	// Background processes always become zombies when they exit.
	// Because the sub-process is now running in the background it will become a zombie,
	// so we must wait for it.
	// Because we run the process in the background, we can Process.Wait for it to get the exit code.
	// Instead, we need to reap it and get the exit code
	pid := process.Pid
	if err := process.Release(); err != nil {
		return err
	}

	for {
		processes, err := ps.Processes()
		if err != nil {
			return err
		}
		found := false
		for _, p := range processes {
			found = found || pid == p.Pid()
		}
		if !found {
			break
		}
		time.Sleep(time.Second)
	}
	for {
		var s syscall.WaitStatus
		wpid, err := syscall.Wait4(-1, &s, syscall.WNOHANG, nil)
		if err != nil {
			return err
		}
		if wpid == pid {
			return errors.NewExitErr(s.ExitStatus())
		}
	}
}
