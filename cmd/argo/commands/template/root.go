package template

import (
	"github.com/spf13/cobra"
)

func NewTemplateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "template",
		Short: "manipulate workflow templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(NewGetCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewCreateCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewLintCommand())
	command.AddCommand(NewUpdateCommand())

	return command
}
