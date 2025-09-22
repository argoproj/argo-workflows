package sync

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type cliUpdateOpts struct {
	key       string // --key
	sizeLimit int32  // --size-limit
}

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts = cliUpdateOpts{}

	command := &cobra.Command{
		Use:     "update",
		Short:   "Update a configmap sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync configmap update my-cm --key my-key --size-limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateSyncLimitCommand(cmd.Context(), args[0], &cliUpdateOpts)
		},
	}

	command.Flags().StringVar(&cliUpdateOpts.key, "key", "", "Key of the sync limit")
	command.Flags().Int32Var(&cliUpdateOpts.sizeLimit, "size-limit", 0, "Size limit of the sync limit")

	ctx := command.Context()
	err := command.MarkFlagRequired("key")
	errors.CheckError(ctx, err)

	err = command.MarkFlagRequired("size-limit")
	errors.CheckError(ctx, err)

	return command
}

func UpdateSyncLimitCommand(ctx context.Context, cmName string, cliOpts *cliUpdateOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient()
	if err != nil {
		return err
	}

	req := &syncpkg.UpdateSyncLimitRequest{
		Name:      cmName,
		Namespace: client.Namespace(ctx),
		Key:       cliOpts.key,
		SizeLimit: cliOpts.sizeLimit,
		Type:      syncpkg.SyncConfigType_CONFIG_MAP,
	}

	resp, err := serviceClient.UpdateSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update sync limit: %v", err)
	}

	fmt.Printf("Updated sync limit for ConfigMap %s from namespace %s with key %s to size limit %d\n", resp.Name, resp.Namespace, resp.Key, resp.SizeLimit)
	return nil
}
