package commands

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
)

func NewDataCommand() *cobra.Command {
	var command = cobra.Command{
		Use:   "data",
		Short: "Process data",
		Run: func(cmd *cobra.Command, args []string) {

			var data v1alpha1.DataTemplate
			log.Info("ARGS:", args)
			err := json.Unmarshal([]byte(args[0]), &data)
			if err != nil {
				log.Fatalf("first argument is not of a WithArtifactPaths type")
			}

			ctx := context.Background()
			err = processData(ctx, &data)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func processData(ctx context.Context, data *v1alpha1.DataTemplate) error {
	wfExecutor := initExecutor()
	return wfExecutor.ProcessData(ctx, data)
}
