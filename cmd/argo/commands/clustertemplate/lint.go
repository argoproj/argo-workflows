package clustertemplate

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
		output string
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of cluster workflow template manifests",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			opts := lint.LintOptions{
				Files:            args,
				DefaultNamespace: client.Namespace(),
				Strict:           strict,
				Printer:          os.Stdout,
			}

			return lint.RunLint(ctx, apiClient, []string{wf.ClusterWorkflowTemplatePlural}, output, false, opts)
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
