package reaper

import (
	"fmt"
	"syscall"
)

func GetKillCommand(s syscall.Signal) []string {
	return ListKillCommands(s)[0]
}

func ListKillCommands(s syscall.Signal) [][]string {
	return [][]string{
		{
			"kill",
			fmt.Sprintf("-%d", s),
			"--",
			"-1", // negative process value signals the process group, rather than just PID 1
		},
		{
			"/bin/sh", // kill is sometimes a built-in (e.g. on Debian), not a script so we must invoke `sh`
			"-c",
			fmt.Sprintf("kill -%d -- -1", s), // negative process value signals the process group, rather than just PID 1
		},
	}
}
