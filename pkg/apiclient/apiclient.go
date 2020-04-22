package apiclient

import (
	"context"
	"fmt"

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
	NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient
	NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient
	NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient
	NewClusterWorkflowTemplateServiceClient() clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient
	NewInfoServiceClient() (infopkg.InfoServiceClient, error)
}

type Opts struct {
	ArgoServer   string
	InstanceID   string
	AuthSupplier func() string
	ClientConfig clientcmd.ClientConfig
}

// DEPRECATED: use NewClientFromOpts
func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	return NewClientFromOpts(Opts{
		ArgoServer:   argoServer,
		AuthSupplier: authSupplier,
		ClientConfig: clientConfig,
	})
}

func NewClientFromOpts(opts Opts) (context.Context, Client, error) {
	if opts.ArgoServer != "" && opts.InstanceID != "" {
		return nil, nil, fmt.Errorf("cannot use instance ID with Argo Server")
	}
	if opts.ArgoServer != "" {
		return newArgoServerClient(opts.ArgoServer, opts.AuthSupplier())
	} else {
		return newArgoKubeClient(opts.ClientConfig, opts.InstanceID)
	}
}
