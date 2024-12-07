package clustertemplate

import (
	"github.com/spf13/cobra"
)

func NewClusterTemplateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "cluster-template",
		Aliases: []string{"cwftmpl", "cwft"},
		Short:   "manipulate cluster workflow templates",
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
