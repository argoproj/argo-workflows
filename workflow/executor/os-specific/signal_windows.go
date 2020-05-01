package os_specific

import (
	"os"
	"syscall"
)

func GetOsSignal() os.Signal {
	return syscall.SIGINT
}
