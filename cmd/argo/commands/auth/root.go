package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "manage authentication settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(NewTokenCommand())
	command.AddCommand(NewSsoCommand())
	return command
}
