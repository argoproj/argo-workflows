package cron

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
)

// NewSuspendCommand returns a new instance of an `argo suspend` command
func NewResumeCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "resume [CRON_WORKFLOW...]",
		Short: "resume zero or more cron workflows",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewCronWorkflowServiceClient()
			namespace := client.Namespace()
			for _, name := range args {
				cronWf, err := serviceClient.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{
					Name:      name,
					Namespace: namespace,
				})
				errors.CheckError(err)
				cronWf.Spec.Suspend = false
				_, err = serviceClient.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{
					Name:         cronWf.Name,
					Namespace:    cronWf.Namespace,
					CronWorkflow: cronWf,
				})
				errors.CheckError(err)
				fmt.Printf("CronWorkflow '%s' resumed\n", name)
			}
		},
	}

	return command
}
