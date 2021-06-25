package os_specific

import (
	"os"
	"syscall"
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
