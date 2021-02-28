package os_specific

import (
	"os"
	"syscall"
)

func GetOsSignal() os.Signal {
	return syscall.SIGUSR2
}

func IsSIGCHLD(s os.Signal) bool { return s == syscall.SIGCHLD }

func Kill(pid int, s syscall.Signal) error {
	return syscall.Kill(pid, s)
}

func Setpgid(a *syscall.SysProcAttr) {
	a.Setpgid = true
}
