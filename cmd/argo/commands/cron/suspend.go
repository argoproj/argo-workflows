package cron

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
)

// NewSuspendCommand returns a new instance of an `argo suspend` command
func NewSuspendCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "suspend CRON_WORKFLOW...",
		Short: "suspend zero or more cron workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			if err != nil {
				return err
			}
			namespace := client.Namespace(ctx)
			for _, name := range args {
				cronWf, err := serviceClient.SuspendCronWorkflow(ctx, &cronworkflowpkg.CronWorkflowSuspendRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					return err
				}
				cronWf.Spec.Suspend = true
				fmt.Printf("CronWorkflow '%s' suspended\n", name)
			}
			return nil
		},
	}
	return command
}
