package db

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type cliCreateOpts struct {
	sizeLimit int32 // --size-limit
}

func NewCreateCommand() *cobra.Command {

	var cliCreateOpts = cliCreateOpts{}

	command := &cobra.Command{
		Use:     "create",
		Short:   "create a db sync limit",
		Args:    cobra.ExactArgs(1),
		Example: `argo sync db create my-key --size-limit 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateSyncLimitCommand(cmd.Context(), args[0], &cliCreateOpts)
		},
	}

	command.Flags().Int32Var(&cliCreateOpts.sizeLimit, "size-limit", 0, "Size limit of the sync limit")

	ctx := command.Context()
	err := command.MarkFlagRequired("size-limit")
	errors.CheckError(ctx, err)

	return command
}

func CreateSyncLimitCommand(ctx context.Context, key string, cliOpts *cliCreateOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient(ctx)
	if err != nil {
		return err
	}

	req := &syncpkg.CreateSyncLimitRequest{
		Key:       key,
		Namespace: client.Namespace(ctx),
		SizeLimit: cliOpts.sizeLimit,
		Type:      syncpkg.SyncConfigType_DATABASE,
	}

	resp, err := serviceClient.CreateSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create sync limit: %v", err)
	}

	fmt.Printf("Database sync limit %s created in namespace %s with size limit %d\n", resp.Name, resp.Namespace, resp.SizeLimit)

	return nil
}
