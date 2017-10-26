package main

import (
	"context"
	"fmt"
	"os"

	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/controller"
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
	argoExecImage string // --argoexec-image
	kubeConfig    string // --kubeconfig
	configMap     string // --configmap
}

var (
	rootArgs rootFlags
)

func init() {
	RootCmd.Flags().StringVar(&rootArgs.kubeConfig, "kubeconfig", "", "Kubernetes config (used when running outside of cluster)")
	RootCmd.Flags().StringVar(&rootArgs.argoExecImage, "argoexec-image", "", "argoexec image to use as container sidecars")
	RootCmd.Flags().StringVar(&rootArgs.configMap, "configmap", "", "Name of K8s configmap to retrieve workflow controller configuration")
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
		panic(err.Error())
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// initialize custom resource using a CustomResourceDefinition if it does not exist
	_, err = workflowclient.CreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}

	// start a controller on instances of our custom resource
	wfController := controller.NewWorkflowController(config)
	if rootArgs.argoExecImage != "" {
		wfController.ArgoExecImage = rootArgs.argoExecImage
	}

	ctx, _ := context.WithCancel(context.Background())
	go wfController.Run(ctx)

	// Wait forever
	select {}
}
