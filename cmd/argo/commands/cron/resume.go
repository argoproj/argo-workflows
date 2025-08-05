package cron

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

// NewResumeCommand returns a new instance of an `argo resume` command
func NewResumeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "resume [CRON_WORKFLOW...]",
		Short: "resume zero or more cron workflows",
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
			for _, name := range args {
				_, err := serviceClient.ResumeCronWorkflow(ctx, &cronworkflowpkg.CronWorkflowResumeRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					return err
				}
				fmt.Printf("CronWorkflow '%s' resumed\n", name)
			}
			return nil
		},
	}

	return command
}
