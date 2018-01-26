package commands

import (
	"os"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/executor"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

var (
	// GlobalArgs hold global CLI flags
	GlobalArgs globalFlags
)

type globalFlags struct {
	podAnnotationsPath string // --pod-annotations
	kubeConfig         string // --kubeconfig
}

func init() {
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.kubeConfig, "kubeconfig", "", "Kubernetes config (used when running outside of cluster)")
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.podAnnotationsPath, "pod-annotations", common.PodMetadataAnnotationsPath, "Pod annotations file from k8s downward API")
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "argoexec is the executor sidecar to workflow containers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

// getClientConfig return rest config, if path not specified, assume in cluster config
func getClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func initExecutor() *executor.WorkflowExecutor {
	podAnnotationsPath := common.PodMetadataAnnotationsPath

	// Use the path specified from the flag
	if GlobalArgs.podAnnotationsPath != "" {
		podAnnotationsPath = GlobalArgs.podAnnotationsPath
	}

	config, err := getClientConfig(GlobalArgs.kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	podName, ok := os.LookupEnv(common.EnvVarPodName)
	if !ok {
		log.Fatalf("Unable to determine pod name from environment variable %s", common.EnvVarPodName)
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		log.Fatalf("Unable to determine pod namespace from environment variable %s", common.EnvVarNamespace)
	}

	wfExecutor := executor.NewExecutor(clientset, podName, namespace, podAnnotationsPath)
	err = wfExecutor.LoadTemplate()
	if err != nil {
		panic(err.Error())
	}
	yamlBytes, _ := yaml.Marshal(&wfExecutor.Template)
	log.Infof("Executor (version: %s) initialized with template:\n%s", argo.GetVersion(), string(yamlBytes))
	return &wfExecutor
}
