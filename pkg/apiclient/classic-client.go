package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/util/help"
)

type classicClient struct {
	versioned.Interface
	kubeClient kubernetes.Interface
}

func newClassicClient(clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	return context.Background(), &classicClient{wfClient, kubeClient}, nil
}

func (a *classicClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &classicWorkflowServiceClient{a.Interface, a.kubeClient}
}

func (a *classicClient) NewCronWorkflowServiceClient() cronworkflow.CronWorkflowServiceClient {
	return &classicCronWorkflowServiceClient{a.Interface}
}

func (a *classicClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, fmt.Errorf("it is impossible to interact with the workflow archive if you are not using the Argo Server, see " + help.CLI)
}
