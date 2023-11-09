package executorplugin

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "executor-plugin",
		Short: "manage executor plugins",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	command.AddCommand(NewBuildCommand())

	return command
}
