package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

func NewLintCommand() *cobra.Command {
	var (
		strict    bool
		lintKinds []string
		output    string
		offline   bool
	)

	allKinds := []string{wf.WorkflowPlural, wf.WorkflowTemplatePlural, wf.CronWorkflowPlural, wf.ClusterWorkflowTemplatePlural}

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of manifests",
		Example: `
# Lint all manifests in a specified directory:

  argo lint ./manifests

# Lint only manifests of Workflows and CronWorkflows from stdin:

  cat manifests.yaml | argo lint --kinds=workflows,cronworkflows -`,
		Run: func(cmd *cobra.Command, args []string) {
			client.Offline = offline
			client.OfflineFiles = args
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			if len(lintKinds) == 0 || strings.Contains(strings.Join(lintKinds, ","), "all") {
				lintKinds = allKinds
			}
			ops := lint.LintOptions{
				Files:            args,
				Strict:           strict,
				DefaultNamespace: client.Namespace(),
				Printer:          os.Stdout,
			}
			lint.RunLint(ctx, apiClient, lintKinds, output, offline, ops)
		},
	}

	command.Flags().StringSliceVar(&lintKinds, "kinds", []string{"all"}, fmt.Sprintf("Which kinds will be linted. Can be: %s", strings.Join(allKinds, "|")))
	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "Perform strict workflow validation")
	command.Flags().BoolVar(&offline, "offline", false, "perform offline linting. For resources referencing other resources, the references will be resolved from the provided args")

	return command
}
