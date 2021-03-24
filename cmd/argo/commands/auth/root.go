package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "manage authentication settings",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewTokenCommand())
	return command
}
