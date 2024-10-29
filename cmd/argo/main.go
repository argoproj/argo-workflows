package main

import (
	"os"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
