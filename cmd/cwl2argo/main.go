package main

import (
	"os"
	"os/exec"

	"github.com/argoproj/argo-workflows/v3/cmd/cwl2argo/commands"
)

func main() {

	err := commands.NewRootCommand().Execute()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() >= 0 {
				os.Exit(exitError.ExitCode())
			} else {
				os.Exit(137) // probably SIGTERM or SIGKILL
			}
		} else {
			println(err.Error())
			os.Exit(64)
		}
	}
}
