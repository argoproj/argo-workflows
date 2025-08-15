package cron

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var all bool

	command := &cobra.Command{
		Use:   "delete [CRON_WORKFLOW... | --all]",
		Short: "delete a cron workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewCronWorkflowServiceClient()
			if err != nil {
				return err
			}
			if all {
				cronWfList, err := serviceClient.ListCronWorkflows(ctx, &cronworkflowpkg.ListCronWorkflowsRequest{
					Namespace: client.Namespace(ctx),
				})
				if err != nil {
					return err
				}
				for _, cronWf := range cronWfList.Items {
					args = append(args, cronWf.Name)
				}
			}
			for _, name := range args {
				_, err := serviceClient.DeleteCronWorkflow(ctx, &cronworkflowpkg.DeleteCronWorkflowRequest{
					Name:      name,
					Namespace: client.Namespace(ctx),
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all cron workflows")
	return command
}
