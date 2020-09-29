package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	cmdutil "github.com/argoproj/argo/util/cmd"
)

// NewVersionCmd returns a new `version` command to be used as a sub-command to root
func NewVersionCommand() *cobra.Command {
	var short bool
	cmd := cobra.Command{
		Use:          "version",
		Short:        "Print version information",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.PrintVersion(CLIName, argo.GetVersion(), short)
			if _, ok := os.LookupEnv("ARGO_SERVER"); ok {
				ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
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
