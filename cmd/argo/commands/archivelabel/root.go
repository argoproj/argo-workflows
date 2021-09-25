package archivelabel

import (
	"github.com/spf13/cobra"
)

func NewArchiveLabelCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "archive-label",
		Short: "manage the workflow archive label",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewListCommand())
	command.AddCommand(NewGetCommand())
	return command
}
