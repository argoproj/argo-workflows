package template

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
		output = common.EnumFlagValue{
			AllowedValues: []string{"pretty", "simple"},
			Value:         "pretty",
		}
	)

	command := &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a file or directory of workflow template manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			opts := lint.LintOptions{
				Files:            args,
				Strict:           strict,
				DefaultNamespace: client.Namespace(),
				Printer:          os.Stdout,
			}
			lint.RunLint(ctx, apiClient, []string{wf.WorkflowTemplatePlural}, output.String(), false, opts)
		},
	}

	command.Flags().VarP(&output, "output", "o", "Linting results output format. "+output.Usage())
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
