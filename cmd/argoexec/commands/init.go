package commands

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/workflow/executor"
	"github.com/argoproj/argo/workflow/util/path"
)

func NewInitCommand() *cobra.Command {
	var command = cobra.Command{
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

	if err := copyEntrypoint(); err != nil {
		wfExecutor.AddError(err)
		return err
	}

	if err := writeTemplate(wfExecutor); err != nil {
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

func writeTemplate(wfExecutor *executor.WorkflowExecutor) error {
	data, err := json.Marshal(wfExecutor.Template)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/var/argo/template", data, 0400) // chmod -r--------
}

func copyEntrypoint() error {
	name, err := path.Search("entrypoint")
	if err != nil {
		return err
	}
	in, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile("/var/argo/entrypoint", os.O_RDWR|os.O_CREATE, 0500) // r-x------
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return err
}
