package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/argoproj/argo/cmd/argoexec/commands"
	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	if err := commands.NewRootCommand().Execute(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			os.Exit(err.ExitCode())
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
