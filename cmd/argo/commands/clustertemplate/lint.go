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
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient()
			lint.RunLint(ctx, apiClient, args, []string{wf.ClusterWorkflowTemplatePlural}, client.Namespace(), output, strict)
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
