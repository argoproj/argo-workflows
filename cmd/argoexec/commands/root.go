package commands

import (
	"github.com/argoproj/argo/util/cmd"
	"github.com/spf13/cobra"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "argoexec is executor sidekick to workflow containers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}
