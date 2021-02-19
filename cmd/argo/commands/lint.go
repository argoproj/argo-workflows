package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
)

func NewLintCommand() *cobra.Command {
	var (
		strict   bool
		allKinds bool
		format   string
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of workflow manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient()
			clients := lint.ServiceClients{
				WorkflowsClient: apiClient.NewWorkflowServiceClient(),
			}
			if allKinds {
				clients.WorkflowTemplatesClient = apiClient.NewWorkflowTemplateServiceClient()
				clients.CronWorkflowsClient = apiClient.NewCronWorkflowServiceClient()
				clients.ClusterWorkflowTemplateClient = apiClient.NewClusterWorkflowTemplateServiceClient()
			}
			fmtr, err := lint.GetFormatter(format)
			errors.CheckError(err)

			res, err := lint.Lint(ctx, &lint.LintOptions{
				ServiceClients:   clients,
				Files:            args,
				Strict:           strict,
				DefaultNamespace: client.Namespace(),
				Formatter:        fmtr,
			})
			errors.CheckError(err)

			fmt.Print(res.Msg())
			if !res.Success {
				os.Exit(1)
			}
		},
	}

	command.Flags().BoolVar(&allKinds, "all-kinds", false, "Lint all kinds, not just workflows")
	command.Flags().StringVar(&format, "format", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "Perform strict workflow validation")

	return command
}
