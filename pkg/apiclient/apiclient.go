package apiclient

import (
	"context"

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

type Opts struct {
	ArgoServerOpts ArgoServerOpts
	AuthSupplier   func() string
	ClientConfig   clientcmd.ClientConfig
}

// DEPRECATED: use NewClientFromOpts
func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	return NewClientFromOpts(Opts{
		ArgoServerOpts: ArgoServerOpts{URL: argoServer},
		AuthSupplier:   authSupplier,
		ClientConfig:   clientConfig,
	})
}

func NewClientFromOpts(opts Opts) (context.Context, Client, error) {
	if opts.ArgoServerOpts.URL != "" {
		return newArgoServerClient(opts.ArgoServerOpts, opts.AuthSupplier())
	} else {
		return newArgoKubeClient(opts.ClientConfig)
	}
}
