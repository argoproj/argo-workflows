package commands

import (
	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	cmdutil "github.com/argoproj/argo/util/cmd"
)

// NewVersionCmd returns a new `version` command to be used as a sub-command to root
func NewVersionCommand() *cobra.Command {
	var short bool
	cmd := cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.PrintVersion(CLIName, argo.GetVersion(), short)
			ctx, apiClient := client.NewAPIClient()
			serviceClient, err := apiClient.NewInfoServiceClient()
			if err == apiclient.NoArgoServerErr {
				return
			}
			errors.CheckError(err)
			serverVersion, err := serviceClient.GetVersion(ctx, &infopkg.GetVersionRequest{})
			errors.CheckError(err)
			cmdutil.PrintVersion("argo-server", *serverVersion, short)
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	return &cmd
}
