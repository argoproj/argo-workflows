package commands

import (
	"context"
	"encoding/json"
	"github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewArtifactsCommand() *cobra.Command {
	var command = cobra.Command{
		Use:   "artifacts",
		Short: "Process artifacts",
		Run: func(cmd *cobra.Command, args []string) {

			var artifacts v1alpha1.WithArtifacts
			err := json.Unmarshal([]byte(args[1]), &artifacts)
			if err != nil {
				log.Fatalf("first argument is not of a WithArtifacts type")
			}

			ctx := context.Background()
			err = processArtifacts(ctx, &artifacts)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func processArtifacts(ctx context.Context, artifacts *v1alpha1.WithArtifacts) error {
	wfExecutor := initExecutor()
	return wfExecutor.ProcessArtifacts(ctx, artifacts)
}

