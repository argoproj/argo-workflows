package commands

import (
	"github.com/spf13/cobra"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// NewInitCommand is the legacy init container: the plan's Prepare phase.
func NewInitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Returned unwrapped: main() reads the exit code with a type assertion.
			return argoexecexecutor.RunHarness(cmd.Context(), clientConfig, varRunArgo, wfexecutor.InitRole)
		},
	}
	return &command
}
