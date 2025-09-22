package sync

import (
	"github.com/spf13/cobra"

	configmap "github.com/argoproj/argo-workflows/v3/cmd/argo/commands/sync/configmap"
)

func NewSyncCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "sync",
		Short: "manage sync limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(configmap.NewConfigmapCommand())

	return command
}
