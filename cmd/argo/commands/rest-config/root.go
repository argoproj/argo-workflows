package rest_config

import (
	"github.com/spf13/cobra"
)

func NewRestConfigCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "rest-config",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewAddCommand())
	command.AddCommand(NewRMCommand())
	return command
}
