package sync

import (
	"github.com/spf13/cobra"
)

func NewSyncCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "sync",
		Short: "manage sync limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(NewCreateCommand())
	command.AddCommand(NewUpdateCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewGetCommand())

	return command
}
