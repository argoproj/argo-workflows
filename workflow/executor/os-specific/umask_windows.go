package os_specific

import "syscall"

func CallUmask(mask int) (oldmask int) {
	// There's no umask in windows.
	return 0
}
