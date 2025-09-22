package sync

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type cliCreateOpts struct {
	key       string // --key
	sizeLimit int32  // --size-limit
}

func NewCreateCommand() *cobra.Command {

	var cliCreateOpts = cliCreateOpts{}

	command := &cobra.Command{
		Use:     "create",
		Short:   "Create a configmap sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync configmap create my-cm --key my-key --size-limit 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateSyncLimitCommand(cmd.Context(), args[0], &cliCreateOpts)
		},
	}

	command.Flags().StringVar(&cliCreateOpts.key, "key", "", "Key of the sync limit")
	command.Flags().Int32Var(&cliCreateOpts.sizeLimit, "size-limit", 0, "Size limit of the sync limit")

	ctx := command.Context()
	err := command.MarkFlagRequired("key")
	errors.CheckError(ctx, err)

	err = command.MarkFlagRequired("size-limit")
	errors.CheckError(ctx, err)

	return command
}

func CreateSyncLimitCommand(ctx context.Context, cmName string, cliOpts *cliCreateOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient()
	if err != nil {
		return err
	}

	req := &syncpkg.CreateSyncLimitRequest{
		Name:      cmName,
		Namespace: client.Namespace(ctx),
		Key:       cliOpts.key,
		SizeLimit: cliOpts.sizeLimit,
		Type:      syncpkg.SyncConfigType_CONFIG_MAP,
	}

	resp, err := serviceClient.CreateSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create sync limit: %v", err)
	}

	fmt.Printf("Configmap sync limit created: %s/%s with key %s and size limit %d\n", resp.Namespace, resp.Name, resp.Key, resp.SizeLimit)

	return nil
}
