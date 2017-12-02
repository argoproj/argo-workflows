package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/argo"
	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/executor"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

var (
	// Global CLI flags
	GlobalArgs globalFlags
)

func init() {
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

type globalFlags struct {
	hostIP             string // --host-ip
	podAnnotationsPath string // --pod-annotations
}

func init() {
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.hostIP, "host-ip", common.EnvVarHostIP, fmt.Sprintf("IP of host. (Default: %s)", common.EnvVarHostIP))
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.podAnnotationsPath, "pod-annotations", common.PodMetadataAnnotationsPath, fmt.Sprintf("Pod annotations fiel from k8s downward API. (Default: %s)", common.PodMetadataAnnotationsPath))
}

func initExecutor() *executor.WorkflowExecutor {
	podAnnotationsPath := common.PodMetadataAnnotationsPath

	// Use the path specified from the flag
	if GlobalArgs.podAnnotationsPath != "" {
		podAnnotationsPath = GlobalArgs.podAnnotationsPath
	}

	var wfTemplate wfv1.Template

	// Read template
	err := GetTemplateFromPodAnnotations(podAnnotationsPath, &wfTemplate)
	if err != nil {
		log.Fatalf("Error getting template %v", err)
	}

	// Initialize in-cluster Kubernetes client
	config, err := rest.InClusterConfig()
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

	// Initialize workflow executor
	wfExecutor := executor.WorkflowExecutor{
		PodName:   podName,
		Template:  wfTemplate,
		ClientSet: clientset,
		Namespace: namespace,
	}
	yamlBytes, _ := yaml.Marshal(&wfExecutor.Template)
	log.Infof("Executor (version: %s) initialized with template:\n%s", argo.FullVersion, string(yamlBytes))
	return &wfExecutor
}
