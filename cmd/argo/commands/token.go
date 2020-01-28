package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	v1 "github.com/argoproj/argo/cmd/argo/commands/client/v1"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the token",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			client, err := v1.GetClient()
			errors.CheckError(err)
			token, err := client.Token()
			errors.CheckError(err)
			fmt.Print(token)
		},
	}
}
