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

type cliGetOpts struct {
	syncType string // --type
	cmName   string // --cm-name
}

func NewGetCommand() *cobra.Command {
	opts := cliGetOpts{}
	command := &cobra.Command{
		Use:   "get",
		Short: "Get a sync limit",
		Args:  cobra.ExactArgs(1),
		Example: `
# Get a database sync limit
	argo sync get my-key --type database

# Get a configmap sync limit
	argo sync get my-key --type configmap --cm-name my-configmap
`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.syncType = strings.ToUpper(opts.syncType)
			return validateFlags(opts.syncType, opts.cmName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetSyncLimitCommand(cmd.Context(), args[0], &opts)
		},
	}

	command.Flags().StringVar(&opts.syncType, "type", "", "Type of sync limit (database or configmap)")
	command.Flags().StringVar(&opts.cmName, "cm-name", "", "ConfigMap name (required if type is configmap)")

	err := command.MarkFlagRequired("type")
	errors.CheckError(command.Context(), err)

	return command
}

func GetSyncLimitCommand(ctx context.Context, key string, cliGetOpts *cliGetOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewSyncServiceClient(ctx)
	if err != nil {
		return err
	}

	req := &syncpkg.GetSyncLimitRequest{
		CmName:    cliGetOpts.cmName,
		Namespace: client.Namespace(ctx),
		Key:       key,
		Type:      syncpkg.SyncConfigType(syncpkg.SyncConfigType_value[cliGetOpts.syncType]),
	}

	resp, err := serviceClient.GetSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get sync limit: %w", err)
	}

	printSyncLimit(resp.Key, resp.CmName, resp.Namespace, resp.Limit, resp.Type)
	return nil
}
