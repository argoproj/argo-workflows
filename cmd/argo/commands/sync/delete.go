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
	var cliDeleteOpts = cliDeleteOpts{}

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
			cliDeleteOpts.syncType = strings.ToUpper(cliDeleteOpts.syncType)
			return validateFlags(cliDeleteOpts.syncType, cliDeleteOpts.cmName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeleteSyncLimitCommand(cmd.Context(), args[0], &cliDeleteOpts)
		},
	}

	command.Flags().StringVar(&cliDeleteOpts.syncType, "type", "", "Type of sync limit (database or configmap)")
	command.Flags().StringVar(&cliDeleteOpts.cmName, "cm-name", "", "ConfigMap name (required if type is configmap)")

	err := command.MarkFlagRequired("type")
	errors.CheckError(command.Context(), err)

	return command
}

func DeleteSyncLimitCommand(ctx context.Context, key string, cliDeleteOpts *cliDeleteOpts) error {
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
		CmName:    cliDeleteOpts.cmName,
		Namespace: namespace,
		Key:       key,
		Type:      syncpkg.SyncConfigType(syncpkg.SyncConfigType_value[cliDeleteOpts.syncType]),
	}

	if _, err := serviceClient.DeleteSyncLimit(ctx, req); err != nil {
		return fmt.Errorf("failed to delete sync limit: %v", err)
	}

	fmt.Printf("Sync limit deleted\n")
	return nil
}
