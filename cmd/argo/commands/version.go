package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
)

// NewVersionCommand returns a new `version` command to be used as a sub-command to root
func NewVersionCommand() *cobra.Command {
	var short bool
	cmd := cobra.Command{
		Use:   "version",
		Short: "print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.PrintVersion(CLIName, argo.GetVersion(), short)
			if _, ok := os.LookupEnv("ARGO_SERVER"); ok {
				ctx, apiClient, err := client.NewAPIClient(cmd.Context())
				if err != nil {
					return err
				}
				serviceClient, err := apiClient.NewInfoServiceClient()
				if err != nil {
					return err
				}
				serverVersion, err := serviceClient.GetVersion(ctx, &infopkg.GetVersionRequest{})
				if err != nil {
					return err
				}
				cmdutil.PrintVersion("argo-server", *serverVersion, short)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	return &cmd
}
