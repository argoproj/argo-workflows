package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "auth",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewTokenCommand())
	return command
}
