package os_specific

import (
	"os"
	"syscall"
	"time"
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

func Wait(pid int) error {
	for {
		var s syscall.WaitStatus
		p, _ := syscall.Wait4(-1, &s, syscall.WNOHANG, nil)
		if p <= 0 {
			time.Sleep(time.Second)
		}
		if pid == p {
			if s.ExitStatus() > 0 {
				return exitErr(s.ExitStatus())
			}
			return nil
		}
	}
}
