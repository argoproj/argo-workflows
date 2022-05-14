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

func ReapZombies() {
	for {
		var s syscall.WaitStatus
		pid, _ := syscall.Wait4(-1, &s, syscall.WNOHANG, nil)
		if pid <= 0 {
			time.Sleep(time.Second)
		}
	}
}
