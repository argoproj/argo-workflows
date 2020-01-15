package archive

import (
	"github.com/spf13/cobra"
)

func NewArchiveCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "archive",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewListCommand())
	command.AddCommand(NewGetCommand())
	command.AddCommand(NewDeleteCommand())
	return command
}
