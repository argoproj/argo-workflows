package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"golang.org/x/term"

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
		output    = common.EnumFlagValue{
			AllowedValues: []string{"pretty", "simple"},
			Value:         "pretty",
		}
		offline bool
	)

	command := &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of manifests",
		Example: `
# Lint all manifests in a specified directory:

  argo lint ./manifests

# Lint only manifests of Workflows and CronWorkflows from stdin:

  cat manifests.yaml | argo lint --kinds=workflows,cronworkflows -`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLint(cmd.Context(), args, offline, lintKinds, output.String(), strict)
		},
	}

	command.Flags().StringSliceVar(&lintKinds, "kinds", []string{"all"}, fmt.Sprintf("Which kinds will be linted. Can be: %s", strings.Join(allKinds, "|")))
	command.Flags().VarP(&output, "output", "o", "Linting results output format. "+output.Usage())
	command.Flags().BoolVar(&strict, "strict", true, "Perform strict workflow validation")
	command.Flags().BoolVar(&offline, "offline", false, "perform offline linting. For resources referencing other resources, the references will be resolved from the provided args")
	command.Flags().BoolVar(&common.NoColor, "no-color", false, "Disable colorized output")

	return command
}

func readStdinToTempFile() (string, error) {
	tmpFile, err := os.CreateTemp("", "stdin_temp_*.yaml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, os.Stdin)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func runLint(ctx context.Context, args []string, offline bool, lintKinds []string, output string, strict bool) error {
	// Handle stdin in offline mode to prevent double-read issue
	// (see https://github.com/argoproj/argo-workflows/issues/12819#issuecomment-2041060032)
	if offline && !term.IsTerminal(int(os.Stdin.Fd())) {
		var tempFile string
		for i, OfflineFile := range args {
			if OfflineFile == "-" {
				// Replace stdin placeholder "-" with a temporary file path in arguments
				file, err := readStdinToTempFile()
				if err != nil {
					return err
				}
				args[i] = file
				tempFile = file
				break
			}
		}
		if tempFile != "" {
			defer os.Remove(tempFile)
		}
	}

	client.Offline = offline
	client.OfflineFiles = args
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
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
	return lint.RunLint(ctx, apiClient, lintKinds, output, offline, ops)
}
