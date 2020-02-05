package apiclient

import (
	"context"

	"k8s.io/client-go/tools/clientcmd"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
)

type Client interface {
	NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error)
	NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient
}

func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	if argoServer != "" {
		return newArgoServerClient(argoServer, authSupplier())
	} else {
		return newClassicClient(clientConfig)
	}
}
