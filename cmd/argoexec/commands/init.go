package commands

import (
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	var command = cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		Run: func(cmd *cobra.Command, args []string) {
			err := loadArtifacts()
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func loadArtifacts() error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError()
	defer stats.LogStats()

	// Download input artifacts
	err := wfExecutor.StageFiles()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.LoadArtifacts()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
