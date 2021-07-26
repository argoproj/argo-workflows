package os_specific

func AllowGrantingAccessToEveryone() {
	// There's no umask in Windows.
	// TODO: figure out how we can allow this in Windows.
}
