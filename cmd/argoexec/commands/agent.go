package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/executor"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/plugins/rpc"
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
			for _, name := range getPluginNames() {
				filename := tokenFilename(name)
				log.WithField("plugin", name).
					WithField("filename", filename).
					Info("creating token file for plugin")
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

func getPluginNames() []string {
	var names []string
	if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarPluginNames)), &names); err != nil {
		log.Fatal(err)
	}
	return names
}

func getPluginAddresses() []string {
	var addresses []string
	if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarPluginAddresses)), &addresses); err != nil {
		log.Fatal(err)
	}
	return addresses
}

func NewAgentMainCommand() *cobra.Command {
	return &cobra.Command{
		Use: "main",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initAgentExecutor().Agent(context.Background())
		},
	}
}

func initAgentExecutor() *executor.AgentExecutor {
	version := argo.GetVersion()
	log.WithFields(log.Fields{"version": version.Version}).Info("Starting Workflow Executor")
	config, err := clientConfig.ClientConfig()
	checkErr(err)

	config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-executor/%s", version.Version, "agent Executor"))

	logs.AddK8SLogTransportWrapper(config) // lets log all request as we should typically do < 5 per pod, so this is will show up problems

	namespace, _, err := clientConfig.Namespace()
	checkErr(err)

	clientSet, err := kubernetes.NewForConfig(config)
	checkErr(err)

	restClient := clientSet.RESTClient()

	workflowName, ok := os.LookupEnv(common.EnvVarWorkflowName)
	if !ok {
		log.Fatalf("Unable to determine workflow name from environment variable %s", common.EnvVarWorkflowName)
	}
	workflowUID, ok := os.LookupEnv(common.EnvVarWorkflowUID)
	if !ok {
		log.Fatalf("Unable to determine workflow Uid from environment variable %s", common.EnvVarWorkflowUID)
	}

	addresses := getPluginAddresses()
	names := getPluginNames()
	plugins := make(map[string]executorplugins.TemplateExecutor, len(names))
	for i, address := range addresses {
		name := names[i]
		filename := tokenFilename(name)
		log.WithField("plugin", name).
			WithField("filename", filename).
			Info("loading token file for plugin")
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		plugins[name] = rpc.New(address, string(data))
	}

	return executor.NewAgentExecutor(clientSet, restClient, config, namespace, workflowName, workflowUID, plugins)
}
