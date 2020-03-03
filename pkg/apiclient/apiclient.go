package apiclient

import (
	"context"

	"k8s.io/client-go/tools/clientcmd"

	cronworkflowpkg "github.com/argoproj/argo/v2/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/v2/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/v2/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/v2/pkg/apiclient/workflowtemplate"
)

type Client interface {
	NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error)
	NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient
	NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient
	NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient
}

func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	if argoServer != "" {
		return newArgoServerClient(argoServer, authSupplier())
	} else {
		return newArgoKubeClient(clientConfig)
	}
}
