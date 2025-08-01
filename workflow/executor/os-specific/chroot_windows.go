package os_specific

func CallChroot() error {
	return nil // no chroot on windows
}
