package sync

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type cliDeleteOpts struct {
	key string // --key
}

func NewDeleteCommand() *cobra.Command {
	var cliDeleteOpts = cliDeleteOpts{}

	command := &cobra.Command{
		Use:     "delete",
		Short:   "Delete a configmap sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync configmap delete my-cm --key my-key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteSyncLimitCommand(cmd.Context(), args[0], &cliDeleteOpts)
		},
	}

	command.Flags().StringVar(&cliDeleteOpts.key, "key", "", "Key of the sync limit")

	err := command.MarkFlagRequired("key")
	errors.CheckError(command.Context(), err)

	return command
}

func DeleteSyncLimitCommand(ctx context.Context, cmName string, cliDeleteOpts *cliDeleteOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient()
	if err != nil {
		return err
	}

	namespace := client.Namespace(ctx)
	req := &syncpkg.DeleteSyncLimitRequest{
		Name:      cmName,
		Namespace: namespace,
		Key:       cliDeleteOpts.key,
		Type:      syncpkg.SyncConfigType_CONFIG_MAP,
	}

	if _, err := serviceClient.DeleteSyncLimit(ctx, req); err != nil {
		return fmt.Errorf("failed to delete sync limit: %v", err)
	}

	fmt.Printf("Deleted sync limit for ConfigMap %s from %s namespace with key %s\n", cmName, namespace, cliDeleteOpts.key)
	return nil
}
