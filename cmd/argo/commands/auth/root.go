package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:          "auth",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}
	command.AddCommand(NewTokenCommand())
	return command
}
