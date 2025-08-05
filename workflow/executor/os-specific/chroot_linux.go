package os_specific

import "syscall"

func CallChroot() error {
	err := syscall.Chroot(".")
	return err
}
