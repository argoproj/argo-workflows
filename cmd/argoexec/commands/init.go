package commands

import (
	"context"

	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			err := loadArtifacts(ctx)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func loadArtifacts(ctx context.Context) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError(ctx)
	defer stats.LogStats()

	if err := wfExecutor.Init(); err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Download input artifacts
	err := wfExecutor.StageFiles()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.LoadArtifacts(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
