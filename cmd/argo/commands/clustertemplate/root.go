package clustertemplate

import (
	"github.com/spf13/cobra"
)

func NewClusterTemplateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "cluster-template",
		Aliases: []string{"cwftmpl", "cwft"},
		Short:   "manipulate cluster workflow templates",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
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
