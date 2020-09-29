package clustertemplate

import (
	"github.com/spf13/cobra"
)

func NewClusterTemplateCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:          "cluster-template",
		Aliases:      []string{"cwftmpl", "cwft"},
		Short:        "manipulate cluster workflow templates",
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
