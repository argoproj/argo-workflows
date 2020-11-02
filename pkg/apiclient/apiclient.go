package apiclient

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
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
	// ArgoServerOpts can be used to connect via an exposed argo API.
	ArgoServerOpts ArgoServerOpts
	// InstanceID can be specified in case multiple argo controllers are running and you want to target a specific one.
	InstanceID string
	// AuthSupplier is used in combination with ArgoServerOpts to specify authentication.
	AuthSupplier func() string
	// DEPRECATED: use `RESTConfigSupplier`
	ClientConfig clientcmd.ClientConfig
	// DEPRECATED: use `RESTConfigSupplier`
	ClientConfigSupplier func() clientcmd.ClientConfig
	// RESTConfigSupplier returns a k8s client-go REST config.
	RESTConfigSupplier func() (*rest.Config, error)
	// Context to use for the generated API client.
	Context context.Context
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

// NewClientFromOpts initializes an argo kubernetes client.
func NewClientFromOpts(opts Opts) (context.Context, Client, error) {
	log.WithField("opts", opts).Debug("Client options")
	if opts.ArgoServerOpts.URL != "" && opts.InstanceID != "" {
		return nil, nil, fmt.Errorf("cannot use instance ID with Argo Server")
	}
	if opts.ArgoServerOpts.URL != "" {
		return newArgoServerClient(opts.ArgoServerOpts, opts.AuthSupplier())
	}

	if opts.ClientConfigSupplier != nil {
		opts.ClientConfig = opts.ClientConfigSupplier()
	}
	if opts.ClientConfig != nil {
		opts.RESTConfigSupplier = func() (*rest.Config, error) { return opts.ClientConfig.ClientConfig() }
	}

	cfg, err := opts.RESTConfigSupplier()
	if err != nil {
		return nil, nil, err
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return newArgoKubeClient(ctx, cfg, instanceid.NewService(opts.InstanceID))
}
