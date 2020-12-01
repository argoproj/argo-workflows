package cluster

import (
	"github.com/spf13/cobra"
)

func NewClusterCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "cluster",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewAddCommand())
	command.AddCommand(NewRMCommand())
	return command
}
