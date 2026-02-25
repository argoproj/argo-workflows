package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4"
	argoexecex "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v4/util/logs"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/plugins/rpc"
)

func NewAgentCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:          "agent",
		SilenceUsage: true, // this prevents confusing usage message being printed on error
	}
	cmd.AddCommand(NewAgentInitCommand())
	cmd.AddCommand(NewAgentMainCommand())
	return &cmd
}

func NewAgentInitCommand() *cobra.Command {
	return &cobra.Command{
		Use: "init",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := logging.RequireLoggerFromContext(ctx)

			for _, name := range getPluginNames(ctx) {
				filename := tokenFilename(name)
				logger.WithField("plugin", name).
					WithField("filename", filename).
					Info(ctx, "creating token file for plugin")
				if err := os.Mkdir(filepath.Dir(filename), 0o770); err != nil {
					return err
				}
				token := rand.String(32) // this could have 26^32 ~= 2 x 10^45  possible values, not guessable in reasonable time
				if err := os.WriteFile(filename, []byte(token), 0o440); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func tokenFilename(name string) string {
	return filepath.Join(common.VarRunArgoPath, name, "token")
}

func getPluginNames(ctx context.Context) []string {
	var names []string
	if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarPluginNames)), &names); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).WithFatal().Error(ctx, "Failed to unmarshal plugin names")
		os.Exit(1)
	}
	return names
}

func getPluginAddresses(ctx context.Context) []string {
	var addresses []string
	if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarPluginAddresses)), &addresses); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).WithFatal().Error(ctx, "Failed to unmarshal plugin addresses")
		os.Exit(1)
	}
	return addresses
}

func NewAgentMainCommand() *cobra.Command {
	return &cobra.Command{
		Use: "main",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return initAgentExecutor(ctx).Agent(ctx)
		},
	}
}

func initAgentExecutor(ctx context.Context) *executor.AgentExecutor {
	version := argo.GetVersion()
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"version": version.Version}).Info(ctx, "Starting Workflow Executor")
	config, err := clientConfig.ClientConfig()
	argoexecex.CheckErr(err)

	config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-executor/%s", version.Version, "agent Executor"))

	logs.AddK8SLogTransportWrapper(ctx, config) // lets log all request as we should typically do < 5 per pod, so this is will show up problems

	namespace, _, err := clientConfig.Namespace()
	argoexecex.CheckErr(err)

	clientSet, err := kubernetes.NewForConfig(config)
	argoexecex.CheckErr(err)

	restClient := clientSet.RESTClient()

	workflowName, workflowNameFound := os.LookupEnv(common.EnvVarWorkflowName)
	labelSelector, ok := os.LookupEnv(common.EnvVarTaskSetLabelSelector)
	if !ok && !workflowNameFound {
		logger.WithFatal().Error(ctx, fmt.Sprintf("Unable to determine label selector or workflow name from environment variables %s and %s", common.EnvVarTaskSetLabelSelector, common.EnvVarWorkflowName))
		os.Exit(1)
	}

	addresses := getPluginAddresses(ctx)
	names := getPluginNames(ctx)
	var plugins []executorplugins.TemplateExecutor
	for i, address := range addresses {
		name := names[i]
		filename := tokenFilename(name)
		logger.WithField("plugin", name).
			WithField("filename", filename).
			Info(ctx, "loading token file for plugin")
		data, err := os.ReadFile(filename)
		if err != nil {
			logger.WithError(err).WithFatal().Error(ctx, "Failed to read token file")
			os.Exit(1)
		}
		plugins = append(plugins, rpc.New(address, string(data)))
	}

	return executor.NewAgentExecutor(clientSet, restClient, config, namespace, labelSelector, workflowName, plugins)
}
