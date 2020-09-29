package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"

	"github.com/argoproj/argo/cmd/argo/commands/client"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "token",
		Short:        "Print the auth token",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				cmd.HelpFunc()(cmd, args)
				return cmdcommon.MissingArgumentsError
			}
			fmt.Println(client.GetAuthString())
			return nil
		},
	}
}
