package cron

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
		output = common.EnumFlagValue{AllowedValues: []string{"pretty", "simple"}, Value: "pretty"}
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of cron workflow manifests",
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
			lint.RunLint(ctx, apiClient, []string{wf.CronWorkflowPlural}, output.String(), false, opts)
		},
	}

	command.Flags().VarP(&output, "output", "o", "Linting results output format. "+output.Usage())
	command.Flags().BoolVar(&strict, "strict", true, "perform strict validation")
	return command
}
