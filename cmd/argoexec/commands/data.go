package commands

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewDataCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "data",
		Short: "Process data",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			err := execData(ctx)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func execData(ctx context.Context) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError(ctx)

	// Create a new empty (placeholder) task result with LabelKeyReportOutputsCompleted set to false.
	wfExecutor.InitializeOutput(ctx)
	defer wfExecutor.FinalizeOutput(ctx) //Ensures the LabelKeyReportOutputsCompleted is set to true.

	err := wfExecutor.Data(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
