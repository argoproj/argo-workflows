package main

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argo/util/cmd"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
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
}

var (
	rootArgs rootFlags
)

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))

	RootCmd.Flags().StringVar(&rootArgs.kubeConfig, "kubeconfig", "", "Kubernetes config (used when running outside of cluster)")
	RootCmd.Flags().StringVar(&rootArgs.configMap, "configmap", common.DefaultWorkflowControllerConfigMap, "Name of K8s configmap to retrieve workflow controller configuration")
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
	config, err := GetClientConfig(rootArgs.kubeConfig)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// initialize custom resource using a CustomResourceDefinition if it does not exist
	_, err = workflowclient.CreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		log.Fatalf("%+v", err)
	}

	// start a controller on instances of our custom resource
	wfController := controller.NewWorkflowController(config, rootArgs.configMap)
	err = wfController.ResyncConfig()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	ctx, _ := context.WithCancel(context.Background())
	go wfController.Run(ctx)

	// Wait forever
	select {}
}
