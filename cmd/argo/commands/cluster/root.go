package cluster

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cluster",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	cmd.AddCommand(newGetProfileCommand())
	cmd.AddCommand(newGetRemoteResourcesCommand())
	return cmd
}
