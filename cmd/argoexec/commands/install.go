package commands

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
)

// NewInstallCommand returns `argoexec install`, the init container of a
// pool pod. It copies argoexec into the shared volume for the pod's main
// container and does nothing else: it reads no template or task env and
// never builds API clients, so it runs with only /var/run/argo mounted.
func NewInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Copy argoexec into the shared volume",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return install(cmd.Context())
		},
	}
}

func install(ctx context.Context) error {
	if err := executor.Install(ctx, emissary.New(varRunArgo), varRunArgo); err != nil {
		return err
	}
	logging.RequireLoggerFromContext(ctx).WithField("dir", varRunArgo).Info(ctx, "Installed argoexec")
	return nil
}
