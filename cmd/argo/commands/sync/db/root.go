package db

import "github.com/spf13/cobra"

func NewDBCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "db",
		Short: "manage db sync limits",
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
