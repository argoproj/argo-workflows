package auth

import (
	"fmt"
	"os"
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the auth token",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
            // To avoid logging this sensitive information in clear text,
            // Replace the print statement with a log statement.
			log.Println("Auth string generated")
	}
}
