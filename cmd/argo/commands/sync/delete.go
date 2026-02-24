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

type cliDeleteOpts struct {
	syncType string // --type
	cmName   string // --cm-name
}

func NewDeleteCommand() *cobra.Command {
	opts := cliDeleteOpts{}

	command := &cobra.Command{
		Use:   "delete",
		Short: "Delete a sync limit",
		Args:  cobra.ExactArgs(1),
		Example: `
# Delete a database sync limit
	argo sync delete my-key --type database

# Delete a configmap sync limit
	argo sync delete my-key --type configmap --cm-name my-configmap
`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.syncType = strings.ToUpper(opts.syncType)
			return validateFlags(opts.syncType, opts.cmName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteSyncLimitCommand(cmd.Context(), args[0], &opts)
		},
	}

	command.Flags().StringVar(&opts.syncType, "type", "", "Type of sync limit (database or configmap)")
	command.Flags().StringVar(&opts.cmName, "cm-name", "", "ConfigMap name (required if type is configmap)")

	err := command.MarkFlagRequired("type")
	errors.CheckError(command.Context(), err)

	return command
}

func DeleteSyncLimitCommand(ctx context.Context, key string, opts *cliDeleteOpts) error {
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
		CmName:    opts.cmName,
		Namespace: namespace,
		Key:       key,
		Type:      syncpkg.SyncConfigType(syncpkg.SyncConfigType_value[opts.syncType]),
	}

	if _, err := serviceClient.DeleteSyncLimit(ctx, req); err != nil {
		return fmt.Errorf("failed to delete sync limit: %w", err)
	}

	fmt.Printf("Sync limit deleted\n")
	return nil
}
