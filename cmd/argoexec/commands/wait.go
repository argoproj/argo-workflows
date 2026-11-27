package commands

import (
	"github.com/spf13/cobra"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// NewWaitCommand is the legacy wait container: everything after main starts.
func NewWaitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "wait",
		Short: "wait for main container to finish and save artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Returned unwrapped: main() reads the exit code with a type assertion.
			return argoexecexecutor.RunHarness(cmd.Context(), clientConfig, varRunArgo, wfexecutor.WaitRole)
		},
	}
	return &command
}
