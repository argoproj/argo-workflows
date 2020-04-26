package apiclient

import (
	"context"

	"k8s.io/client-go/tools/clientcmd"

	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
)

type Client interface {
	NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error)
	NewWorkflowServiceClient() (workflowpkg.WorkflowServiceClient, error)
	NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error)
	NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error)
	NewClusterWorkflowTemplateServiceClient() (clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient, error)
	NewInfoServiceClient() (infopkg.InfoServiceClient, error)
}

type Opts struct {
	Offline        bool
	ArgoServerOpts ArgoServerOpts
	AuthSupplier   func() string
	// DEPRECATED: use ClientConfigSupplier
	ClientConfig         clientcmd.ClientConfig
	ClientConfigSupplier func() clientcmd.ClientConfig
}

// DEPRECATED: use NewClientFromOpts
func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	opts := Opts{
		ArgoServerOpts: ArgoServerOpts{URL: argoServer},
		AuthSupplier:   authSupplier,
		ClientConfigSupplier: func() clientcmd.ClientConfig {
			return clientConfig
		},
	}
	return NewClientFromOpts(opts)
}

func NewClientFromOpts(opts Opts) (context.Context, Client, error) {
	if opts.Offline {
		return newOfflineClient()
	} else if opts.ArgoServerOpts.URL != "" {
		return newArgoServerClient(opts.ArgoServerOpts, opts.AuthSupplier())
	} else {
		return newArgoKubeClient(opts.ClientConfigSupplier())
	}
}
