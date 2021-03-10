package reaper

import (
	"fmt"
	"syscall"
)

func GetKillCommand(s syscall.Signal) []string {
	return []string{
		"kill", // no need for /bin/sh
		fmt.Sprintf("-%d", s),
		"--",
		"-1", // negative process value signals the process group, rather than just PID 1
	}
}
