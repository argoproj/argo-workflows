package db

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
)

func NewGetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "get",
		Short:   "get a db sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync db get my-key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetSyncLimitCommand(cmd.Context(), args[0])
		},
	}

	return command
}

func GetSyncLimitCommand(ctx context.Context, key string) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient(ctx)
	if err != nil {
		return err
	}

	req := &syncpkg.GetSyncLimitRequest{
		Type:      syncpkg.SyncConfigType_DATABASE,
		Key:       key,
		Namespace: client.Namespace(ctx),
	}

	resp, err := serviceClient.GetSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get sync limit: %v", err)
	}

	fmt.Printf("Database sync limit %s in namespace %s is %d\n", resp.Key, resp.Namespace, resp.SizeLimit)

	return nil
}
