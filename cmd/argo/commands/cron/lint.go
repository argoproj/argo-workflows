package cron

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
		Short: "validate files or directories of cron workflow manifests",
		Example: `# Validate cron workflow manifest

  argo cron lint FILE1 FILE2
`,
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
			lint.RunLint(ctx, apiClient, []string{wf.CronWorkflowPlural}, output, false, opts)
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "perform strict validation")
	return command
}
