package commands

import (
	"github.com/argoproj/argo/util/cmd"
	"github.com/spf13/cobra"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo"
)

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "argo is the command line interface to Argo",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}
