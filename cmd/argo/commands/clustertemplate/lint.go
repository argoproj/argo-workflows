package clustertemplate

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
		strict bool
		format string
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of cluster workflow template manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient()
			fmtr, err := lint.GetFormatter(format)
			errors.CheckError(err)

			res, err := lint.Lint(ctx, &lint.LintOptions{
				ServiceClients: lint.ServiceClients{
					ClusterWorkflowTemplateClient: apiClient.NewClusterWorkflowTemplateServiceClient(),
				},
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

	command.Flags().StringVar(&format, "format", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
