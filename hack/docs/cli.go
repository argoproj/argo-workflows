package main

import (
	"github.com/spf13/cobra/doc"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands"
)

func generateCLIDocs() {
	println("generating docs/cli")
	cmd := commands.NewCommand()
	cmd.DisableAutoGenTag = true
	err := doc.GenMarkdownTree(cmd, "docs/cli")
	if err != nil {
		panic(err)
	}
}
