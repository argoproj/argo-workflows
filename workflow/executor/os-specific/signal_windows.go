package os_specific

import (
	"fmt"
	"os"
	"syscall"
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
		var errStr string
		if err != nil {
			errStr = err.Error()
		} else {
			errStr = "<nil>"
		}

		return fmt.Errorf("exit with non-zero code. exit-code: %d, error:%s", stat.ExitCode(), errStr)
	}
	return err
}
