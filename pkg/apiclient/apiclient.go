package apiclient

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/util/instanceid"
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
	ArgoServerOpts ArgoServerOpts
	InstanceID     string
	AuthSupplier   func() string
	// DEPRECATED: use `ClientConfigSupplier`
	ClientConfig         clientcmd.ClientConfig
	ClientConfigSupplier func() clientcmd.ClientConfig
}

func (o Opts) String() string {
	return fmt.Sprintf("(argoServerOpts=%v,instanceID=%v)", o.ArgoServerOpts, o.InstanceID)
}

// DEPRECATED: use NewClientFromOpts
func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	return NewClientFromOpts(Opts{
		ArgoServerOpts: ArgoServerOpts{URL: argoServer},
		AuthSupplier:   authSupplier,
		ClientConfigSupplier: func() clientcmd.ClientConfig {
			return clientConfig
		},
	})
}

func NewClientFromOpts(opts Opts) (context.Context, Client, error) {
	log.WithField("opts", opts).Debug("Client options")
	if opts.ArgoServerOpts.URL != "" && opts.InstanceID != "" {
		return nil, nil, fmt.Errorf("cannot use instance ID with Argo Server")
	}
	if opts.ArgoServerOpts.HTTP1 {
		return newHTTP1Client(opts.ArgoServerOpts.GetURL(), opts.AuthSupplier())
	} else if opts.ArgoServerOpts.URL != "" {
		return newArgoServerClient(opts.ArgoServerOpts, opts.AuthSupplier())
	} else {
		if opts.ClientConfigSupplier != nil {
			opts.ClientConfig = opts.ClientConfigSupplier()
		}
		return newArgoKubeClient(opts.ClientConfig, instanceid.NewService(opts.InstanceID))
	}
}
