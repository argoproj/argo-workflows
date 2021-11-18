package plugin

import (
	"github.com/spf13/cobra"
)

func NewPluginCommand() *cobra.Command {
	command := &cobra.Command{
		Use: "plugin",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	command.AddCommand(NewBuildCommand())

	return command
}
