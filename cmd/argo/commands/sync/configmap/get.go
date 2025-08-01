package sync

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type cliGetOpts struct {
	key string // --key
}

func NewGetCommand() *cobra.Command {
	var cliGetOpts = cliGetOpts{}
	command := &cobra.Command{
		Use:     "get",
		Short:   "Get a configmap sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync get configmap my-cm --key my-key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetSyncLimitCommand(cmd.Context(), args[0], &cliGetOpts)
		},
	}

	command.Flags().StringVar(&cliGetOpts.key, "key", "", "Key of the sync limit")

	err := command.MarkFlagRequired("key")
	errors.CheckError(command.Context(), err)

	return command
}

func GetSyncLimitCommand(ctx context.Context, cmName string, cliGetOpts *cliGetOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient()
	if err != nil {
		return err
	}

	req := &syncpkg.GetSyncLimitRequest{
		Name:      cmName,
		Namespace: client.Namespace(ctx),
		Key:       cliGetOpts.key,
		Type:      syncpkg.SyncConfigType_CONFIG_MAP,
	}

	resp, err := serviceClient.GetSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get sync limit: %v", err)
	}

	fmt.Printf("Sync Configmap name: %s\nNamespace: %s\nKey: %s\nSize Limit: %d\n", resp.Name, resp.Namespace, resp.Key, resp.SizeLimit)
	return nil
}
