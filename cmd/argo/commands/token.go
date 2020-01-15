package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use: "token",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			fmt.Println(client.GetBearerToken())
		},
	}
}
