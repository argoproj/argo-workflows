package sync

import (
	"github.com/spf13/cobra"
)

func NewConfigmapCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "configmap",
		Aliases: []string{"cm"},
		Short:   "manage configmap sync limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(NewCreateCommand())
	command.AddCommand(NewGetCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewUpdateCommand())

	return command
}
