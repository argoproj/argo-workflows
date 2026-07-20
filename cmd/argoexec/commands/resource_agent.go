package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4"
	argoexecex "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/logs"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/tracing"
)

// NewResourceAgentCommand returns the resource agent command.
func NewResourceAgentCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "resource-agent",
		SilenceUsage: true, // this prevents confusing usage message being printed on error
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return initResourceAgentExecutor(ctx).Agent(ctx)
		},
	}
}

func initResourceAgentExecutor(ctx context.Context) *executor.ResourceAgentExecutor {
	version := argo.GetVersion()
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"version": version.Version}).Info(ctx, "Starting Resource Agent")
	config, err := clientConfig.ClientConfig()
	argoexecex.CheckErr(err)

	config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-executor/%s", version.Version, "resource-agent Executor"))

	logs.AddK8SLogTransportWrapper(ctx, config)
	tracing.AddTracingTransportWrapper(ctx, config)

	namespace, _, err := clientConfig.Namespace()
	argoexecex.CheckErr(err)

	clientSet, err := kubernetes.NewForConfig(config)
	argoexecex.CheckErr(err)

	workflowName, ok := os.LookupEnv(common.EnvVarWorkflowName)
	if !ok {
		logger.WithFatal().Error(ctx, fmt.Sprintf("Unable to determine workflow name from environment variable %s", common.EnvVarWorkflowName))
		os.Exit(1)
	}
	workflowUID, ok := os.LookupEnv(common.EnvVarWorkflowUID)
	if !ok {
		logger.WithFatal().Error(ctx, fmt.Sprintf("Unable to determine workflow Uid from environment variable %s", common.EnvVarWorkflowUID))
		os.Exit(1)
	}

	return executor.NewResourceAgentExecutor(clientSet, config, namespace, workflowName, workflowUID)
}
