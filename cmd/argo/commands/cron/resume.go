package cron

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

// NewResumeCommand returns a new instance of an `argo resume` command
func NewResumeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "resume [CRON_WORKFLOW...]",
		Short: "resume zero or more cron workflows",
		Example: `# Resume a cron workflow

  argo cron resume my-cron-workflow
`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			errors.CheckError(err)
			namespace := client.Namespace()
			for _, name := range args {
				_, err := serviceClient.ResumeCronWorkflow(ctx, &cronworkflowpkg.CronWorkflowResumeRequest{
					Name:      name,
					Namespace: namespace,
				})
				errors.CheckError(err)
				fmt.Printf("CronWorkflow '%s' resumed\n", name)
			}
		},
	}

	return command
}
