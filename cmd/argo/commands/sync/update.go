package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v4/util/errors"
)

type cliUpdateOpts struct {
	limit    int32  // --limit
	syncType string // --type
	cmName   string // --cm-name
}

func NewUpdateCommand() *cobra.Command {
	opts := cliUpdateOpts{}

	command := &cobra.Command{
		Use:   "update",
		Short: "Update a configmap sync limit",
		Args:  cobra.ExactArgs(1),
		Example: `
# Update a database sync limit
	argo sync update my-key --type database --size-limit 20

# Update a configmap sync limit
	argo sync update my-key --type configmap --cm-name my-configmap --size-limit 20
`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.syncType = strings.ToUpper(opts.syncType)
			return validateFlags(opts.syncType, opts.cmName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateSyncLimitCommand(cmd.Context(), args[0], &opts)
		},
	}

	command.Flags().StringVar(&opts.cmName, "cm-name", "", "ConfigMap name (required if type is configmap)")
	command.Flags().Int32Var(&opts.limit, "limit", 0, "Limit of the sync limit")
	command.Flags().StringVar(&opts.syncType, "type", "", "Type of sync limit (database or configmap)")

	ctx := command.Context()
	err := command.MarkFlagRequired("type")
	errors.CheckError(ctx, err)

	err = command.MarkFlagRequired("limit")
	errors.CheckError(ctx, err)

	return command
}

func UpdateSyncLimitCommand(ctx context.Context, key string, cliOpts *cliUpdateOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient(ctx)
	if err != nil {
		return err
	}

	req := &syncpkg.UpdateSyncLimitRequest{
		CmName:    cliOpts.cmName,
		Namespace: client.Namespace(ctx),
		Key:       key,
		Limit:     cliOpts.limit,
		Type:      syncpkg.SyncConfigType(syncpkg.SyncConfigType_value[cliOpts.syncType]),
	}

	resp, err := serviceClient.UpdateSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update sync limit: %w", err)
	}

	fmt.Printf("Sync limit updated\n")
	printSyncLimit(resp.Key, resp.CmName, resp.Namespace, resp.Limit, resp.Type)
	return nil
}
