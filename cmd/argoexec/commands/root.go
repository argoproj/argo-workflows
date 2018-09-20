package commands

import (
	"os"

	"github.com/argoproj/pkg/kube/cli"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/executor"
	"github.com/argoproj/argo/workflow/executor/docker"
	"github.com/argoproj/argo/workflow/executor/k8sapi"
	"github.com/argoproj/argo/workflow/executor/kubelet"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

var (
	// GlobalArgs hold global CLI flags
	GlobalArgs globalFlags

	clientConfig clientcmd.ClientConfig
)

type globalFlags struct {
	podAnnotationsPath string // --pod-annotations
}

func init() {
	clientConfig = cli.AddKubectlFlagsToCmd(RootCmd)
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

func initExecutor() *executor.WorkflowExecutor {
	podAnnotationsPath := common.PodMetadataAnnotationsPath

	// Use the path specified from the flag
	if GlobalArgs.podAnnotationsPath != "" {
		podAnnotationsPath = GlobalArgs.podAnnotationsPath
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	namespace, _, err := clientConfig.Namespace()
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

	var cre executor.ContainerRuntimeExecutor
	switch os.Getenv(common.EnvVarContainerRuntimeExecutor) {
	case common.ContainerRuntimeExecutorK8sAPI:
		cre, err = k8sapi.NewK8sAPIExecutor(clientset, config, podName, namespace)
		if err != nil {
			panic(err.Error())
		}
	case common.ContainerRuntimeExecutorKubelet:
		cre, err = kubelet.NewKubeletExecutor()
		if err != nil {
			panic(err.Error())
		}
	default:
		cre, err = docker.NewDockerExecutor()
		if err != nil {
			panic(err.Error())
		}
	}
	wfExecutor := executor.NewExecutor(clientset, podName, namespace, podAnnotationsPath, cre)
	err = wfExecutor.LoadTemplate()
	if err != nil {
		panic(err.Error())
	}

	yamlBytes, _ := yaml.Marshal(&wfExecutor.Template)
	vers := argo.GetVersion()
	log.Infof("Executor (version: %s, build_date: %s) initialized with template:\n%s", vers, vers.BuildDate, string(yamlBytes))
	return &wfExecutor
}
