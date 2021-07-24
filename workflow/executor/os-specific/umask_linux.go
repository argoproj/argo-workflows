package os_specific

import "syscall"

func CallUmask(mask int) (oldmask int) {
	return syscall.Umask(mask)
}
