package commands

import (
	"context"
	"time"

	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewWaitCommand() *cobra.Command {
	var command = cobra.Command{
		Use:   "wait",
		Short: "wait for main container to finish and save artifacts",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			err := waitContainer(ctx)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func waitContainer(ctx context.Context) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError(ctx) // Must be placed at the bottom of defers stack.
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	defer func() {
		if err := wfExecutor.Close(ctx); err != nil {
			wfExecutor.AddError(err)
		}
		if err := wfExecutor.KillSidecars(ctx); err != nil {
			wfExecutor.AddError(err)
		}
	}()

	err := wfExecutor.Wait(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		// do not return here so we can still try to kill sidecars & save outputs
	}
	return err
}
