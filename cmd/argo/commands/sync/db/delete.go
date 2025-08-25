package db

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
)

func NewDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "delete",
		Short:   "delete a db sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync db delete my-key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteSyncLimitCommand(cmd.Context(), args[0])
		},
	}

	return command
}

func DeleteSyncLimitCommand(ctx context.Context, key string) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient(ctx)
	if err != nil {
		return err
	}

	namespace := client.Namespace(ctx)
	req := &syncpkg.DeleteSyncLimitRequest{
		Type:      syncpkg.SyncConfigType_DATABASE,
		Key:       key,
		Namespace: namespace,
	}

	_, err = serviceClient.DeleteSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete sync limit: %v", err)
	}

	fmt.Printf("Database sync limit %s from namespace %s is deleted\n", key, namespace)

	return nil
}
