package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the auth token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			authString, err := client.GetAuthString()
			if err != nil {
				return err
			}
			fmt.Println(authString)
			return nil
		},
	}
}
