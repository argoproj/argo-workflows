package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the auth token",
		Example: `# Print the auth token

  argo auth token`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			fmt.Println(client.GetAuthString())
		},
	}
}
