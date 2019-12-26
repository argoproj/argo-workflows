package history

import (
	"github.com/spf13/cobra"
)

func NewHistoryCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "history",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewListCommand())
	return command
}
