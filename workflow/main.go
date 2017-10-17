package main

import (
	"context"
	"flag"
	"fmt"

	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/controller"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// return rest config, if path not specified assume in cluster config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func main() {

	kubeconf := flag.String("kubeconf", "admin.conf", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	config, err := GetClientConfig(*kubeconf)
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

	ctx, _ := context.WithCancel(context.Background())
	go wfController.Run(ctx)

	// List all Workflow objects
	items, err := wfController.WorkflowClient.ListWorkflows(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("List:\n%s\n", items)

	// Wait forever
	select {}
}
