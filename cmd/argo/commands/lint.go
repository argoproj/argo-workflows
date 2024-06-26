package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

var allKinds = []string{wf.WorkflowPlural, wf.WorkflowTemplatePlural, wf.CronWorkflowPlural, wf.ClusterWorkflowTemplatePlural}

func NewLintCommand() *cobra.Command {
	var (
		strict    bool
		lintKinds []string
		output    string
		offline   bool
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of manifests",
		Example: `
# Lint all manifests in a specified directory:

  argo lint ./manifests

# Lint only manifests of Workflows and CronWorkflows from stdin:

  cat manifests.yaml | argo lint --kinds=workflows,cronworkflows -`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			runLint(cmd.Context(), args, offline, lintKinds, output, strict)
		},
	}

	command.Flags().StringSliceVar(&lintKinds, "kinds", []string{"all"}, fmt.Sprintf("Which kinds will be linted. Can be: %s", strings.Join(allKinds, "|")))
	command.Flags().StringVarP(&output, "output", "o", "pretty", "Linting results output format. One of: pretty|simple")
	command.Flags().BoolVar(&strict, "strict", true, "Perform strict workflow validation")
	command.Flags().BoolVar(&offline, "offline", false, "perform offline linting. For resources referencing other resources, the references will be resolved from the provided args")
	command.Flags().BoolVar(&common.NoColor, "no-color", false, "Disable colorized output")

	return command
}

func runLint(ctx context.Context, args []string, offline bool, lintKinds []string, output string, strict bool) {
	client.Offline = offline
	client.OfflineFiles = args
	ctx, apiClient := client.NewAPIClient(ctx)

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
}
