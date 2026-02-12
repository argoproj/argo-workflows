package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type cliUpdateOpts struct {
	limit    int32  // --limit
	syncType string // --type
	cmName   string // --cm-name
}

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts = cliUpdateOpts{}

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
			cliUpdateOpts.syncType = strings.ToUpper(cliUpdateOpts.syncType)
			return validateFlags(cliUpdateOpts.syncType, cliUpdateOpts.cmName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateSyncLimitCommand(cmd.Context(), args[0], &cliUpdateOpts)
		},
	}

	command.Flags().StringVar(&cliUpdateOpts.cmName, "cm-name", "", "ConfigMap name (required if type is configmap)")
	command.Flags().Int32Var(&cliUpdateOpts.limit, "limit", 0, "Limit of the sync limit")
	command.Flags().StringVar(&cliUpdateOpts.syncType, "type", "", "Type of sync limit (database or configmap)")

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
