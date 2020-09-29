package cron

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
)

// NewSuspendCommand returns a new instance of an `argo suspend` command
func NewSuspendCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:          "suspend CRON_WORKFLOW...",
		Short:        "suspend zero or more cron workflows",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewCronWorkflowServiceClient()
			namespace := client.Namespace()
			for _, name := range args {
				cronWf, err := serviceClient.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					return err
				}
				cronWf.Spec.Suspend = true
				_, err = serviceClient.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{
					Name:         name,
					Namespace:    namespace,
					CronWorkflow: cronWf,
				})
				if err != nil {
					return err
				}
				fmt.Printf("CronWorkflow '%s' suspended\n", name)
			}
			return nil
		},
	}
	return command
}
