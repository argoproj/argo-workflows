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

type cliCreateOpts struct {
	limit    int32  // --limit
	syncType string // --type
	cmName   string // --cm-name
}

func NewCreateCommand() *cobra.Command {

	var cliCreateOpts = cliCreateOpts{}

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a sync limit",
		Args:  cobra.ExactArgs(1),
		Example: `
# Create a database sync limit:
	argo sync create my-key --type database --limit 10
		
# Create a configmap sync limit:
	argo sync create my-key --type configmap --cm-name my-configmap --limit 10
`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			cliCreateOpts.syncType = strings.ToUpper(cliCreateOpts.syncType)
			return validateFlags(cliCreateOpts.syncType, cliCreateOpts.cmName)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateSyncLimitCommand(cmd.Context(), args[0], &cliCreateOpts)
		},
	}

	command.Flags().Int32Var(&cliCreateOpts.limit, "limit", 0, "Sync limit")
	command.Flags().StringVar(&cliCreateOpts.syncType, "type", "", "Type of sync limit (database or configmap)")
	command.Flags().StringVar(&cliCreateOpts.cmName, "cm-name", "", "ConfigMap name (required if type is configmap)")

	ctx := command.Context()

	err := command.MarkFlagRequired("limit")
	errors.CheckError(ctx, err)

	err = command.MarkFlagRequired("type")
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
		CmName:    cliOpts.cmName,
		Namespace: client.Namespace(ctx),
		Key:       key,
		Limit:     cliOpts.limit,
		Type:      syncpkg.SyncConfigType(syncpkg.SyncConfigType_value[cliOpts.syncType]),
	}

	resp, err := serviceClient.CreateSyncLimit(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create sync limit: %v", err)
	}

	fmt.Printf("Sync limit created\n")
	printSyncLimit(resp.Key, resp.CmName, resp.Namespace, resp.Limit, resp.Type)

	return nil
}
