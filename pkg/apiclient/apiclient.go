package apiclient

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
)

type Client interface {
	NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error)
	NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient
	NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient
	NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient
	NewClusterWorkflowTemplateServiceClient() clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient
}

// DEPRECATED: use NewClientFromOpts
func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	if argoServer != "" {
		return newArgoServerClient(ArgoServerOpts{
			URL: argoServer,
		}, authSupplier())
	} else {
		return newArgoKubeClient(clientConfig)
	}
}

func NewClientFromOpts(opts ArgoServerOpts, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	log.WithField("opts", opts).Debug("New client")
	if opts.URL != "" {
		return newArgoServerClient(opts, authSupplier())
	} else {
		return newArgoKubeClient(clientConfig)
	}
}
