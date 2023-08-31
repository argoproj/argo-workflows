package os_specific

import (
	"os"
	"syscall"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

var (
	Term = os.Interrupt
)

func CanIgnoreSignal(s os.Signal) bool {
	return false
}

func Kill(pid int, s syscall.Signal) error {
	if pid < 0 {
		pid = -pid // // we cannot kill a negative process on windows
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(s)
}

func Setpgid(a *syscall.SysProcAttr) {
	// this does not exist on windows
}

func Wait(process *os.Process) error {
	stat, err := process.Wait()
	if stat.ExitCode() != 0 {
		return errors.NewExitErr(stat.ExitCode())
	}
	return err
}
