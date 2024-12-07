package cron

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

func NewGetCommand() *cobra.Command {
	var output = common.NewPrintWorkflowOutputValue("")

	command := &cobra.Command{
		Use:   "get CRON_WORKFLOW...",
		Short: "display details about a cron workflow",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			if err != nil {
				return err
			}
			namespace := client.Namespace()

			for _, arg := range args {
				cronWf, err := serviceClient.GetCronWorkflow(ctx, &cronworkflow.GetCronWorkflowRequest{
					Name:      arg,
					Namespace: namespace,
				})
				if err != nil {
					return err
				}
				printCronWorkflow(ctx, cronWf, output.String())
			}
			return nil
		},
	}

	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	return command
}
