package cron

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/pkg/errors"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

func NewGetCommand() *cobra.Command {
	var output = common.NewPrintWorkflowOutputValue("")

	command := &cobra.Command{
		Use:   "get CRON_WORKFLOW...",
		Short: "display details about a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			errors.CheckError(err)
			namespace := client.Namespace()

			for _, arg := range args {
				cronWf, err := serviceClient.GetCronWorkflow(ctx, &cronworkflow.GetCronWorkflowRequest{
					Name:      arg,
					Namespace: namespace,
				})
				errors.CheckError(err)
				printCronWorkflow(cronWf, output.String())
			}
		},
	}

	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	return command
}
