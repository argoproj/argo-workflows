package template

import (
	"github.com/spf13/cobra"
)

func NewTemplateCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:          "template",
		Short:        "manipulate workflow templates",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}

	command.AddCommand(NewGetCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewCreateCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewLintCommand())

	return command
}
