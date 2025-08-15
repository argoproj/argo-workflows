package osspecific

func CallChroot() error {
	return nil // no chroot on windows
}
