package osspecific

import "syscall"

func CallChroot() error {
	err := syscall.Chroot(".")
	return err
}
