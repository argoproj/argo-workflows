package osspecific

import "syscall"

func AllowGrantingAccessToEveryone() {
	// default umask can be 022
	// setting umask as 0 allow granting write access to other non-root users
	syscall.Umask(0)
}
