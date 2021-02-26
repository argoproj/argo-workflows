package template

import (
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
		output string
	)

	command := &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a file or directory of workflow template manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient()
			fmtr, err := lint.GetFormatter(output)
			errors.CheckError(err)

			res, err := lint.Lint(ctx, &lint.LintOptions{
				ServiceClients: lint.ServiceClients{
					WorkflowTemplatesClient: apiClient.NewWorkflowTemplateServiceClient(),
				},
				Files:            args,
				Strict:           strict,
				DefaultNamespace: client.Namespace(),
				Formatter:        fmtr,
				Output:           os.Stdout,
			})
			errors.CheckError(err)

			if !res.Success {
				os.Exit(1)
			}
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output output. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
