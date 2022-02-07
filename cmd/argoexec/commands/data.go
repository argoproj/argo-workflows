package commands

import (
	"context"

	"github.com/spf13/cobra"
)

func NewDataCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "data",
		Short: "Process data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			wfExecutor := initExecutor()
			return wfExecutor.Data(ctx)
		},
	}
	return &command
}

// ResourceMetadata describe resources of the workflow object condition
type ResourceMetadata struct {
	Namespace string
	Name      string
	selfLink  string
	err       error
}
