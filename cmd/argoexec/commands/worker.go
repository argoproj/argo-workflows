package commands

import (
	"github.com/spf13/cobra"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
)

// NewWorkerCommand returns `argoexec worker`, the worker container of a hot
// worker pool pod. It claims the pool's tasks one at a time, stages each
// for the pod's looping main container (`argoexec emissary --loop`) and
// reports its result, and on SIGTERM stops taking tasks.
func NewWorkerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run a hot worker pool's tasks through the pool pod's main container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return argoexecexecutor.RunWorker(cmd.Context(), clientConfig)
		},
	}
}
