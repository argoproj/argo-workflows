package main

import (
	"context"
	"fmt"
	"os"
	"time"

	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/util/stats"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// load the oidc plugin (required to authenticate with OpenID Connect).
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "workflow-controller"
)

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "workflow-controller is the controller to operate on workflows",
	Run:   Run,
}

type rootFlags struct {
	kubeConfig string // --kubeconfig
	configMap  string // --configmap
	logLevel   string // --loglevel
}

var (
	rootArgs rootFlags
)

func init() {
	RootCmd.AddCommand(cmdutil.NewVersionCmd(CLIName))

	RootCmd.Flags().StringVar(&rootArgs.kubeConfig, "kubeconfig", "", "Kubernetes config (used when running outside of cluster)")
	RootCmd.Flags().StringVar(&rootArgs.configMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "Name of K8s configmap to retrieve workflow controller configuration")
	RootCmd.Flags().StringVar(&rootArgs.logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
}

// GetClientConfig return rest config, if path not specified, assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Run(cmd *cobra.Command, args []string) {
	cmdutil.SetLogLevel(rootArgs.logLevel)
	stats.RegisterStackDumper()
	stats.StartStatsTicker(5 * time.Minute)

	config, err := GetClientConfig(rootArgs.kubeConfig)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	config.Burst = 30
	config.QPS = 20.0

	kubeclientset := kubernetes.NewForConfigOrDie(config)
	wflientset := wfclientset.NewForConfigOrDie(config)

	// start a controller on instances of our custom resource
	wfController := controller.NewWorkflowController(config, kubeclientset, wflientset, rootArgs.configMap)
	err = wfController.ResyncConfig()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go wfController.Run(ctx, 8, 8)

	// Wait forever
	select {}
}
