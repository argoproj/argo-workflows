package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type kubeClient struct {
	versioned.Interface
}

func newKubeClient(clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	return context.Background(), &kubeClient{wfClient}, nil
}

func (a *kubeClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &kubeWorkflowServiceClient{a.Interface}
}

func (a *kubeClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, fmt.Errorf("it is impossible to interact with the archive if you are not using the Argo Server")
}
