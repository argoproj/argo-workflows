package commands

import (
	"github.com/spf13/cobra"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

func NewResourceCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "resource (get|create|apply|delete) MANIFEST",
		Short: "update a resource and wait for resource conditions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Returned unwrapped: main() reads the exit code with a type assertion.
			return argoexecexecutor.RunHarness(cmd.Context(), clientConfig, varRunArgo, wfexecutor.ResourceContainerRole)
		},
	}
	return &command
}
