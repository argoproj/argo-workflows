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
			err := processData(ctx)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func processData(ctx context.Context) error {
	wfExecutor := initExecutor()
	return wfExecutor.Data(ctx)
}
