package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type Client interface {
	NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error)
	NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient
	NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error)
	NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error)
	NewClusterWorkflowTemplateServiceClient() (clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient, error)
	NewInfoServiceClient() (infopkg.InfoServiceClient, error)
}

type Opts struct {
	ArgoServerOpts ArgoServerOpts
	ArgoKubeOpts   ArgoKubeOpts
	InstanceID     string
	AuthSupplier   func() string
	// DEPRECATED: use `ClientConfigSupplier`
	ClientConfig         clientcmd.ClientConfig
	ClientConfigSupplier func() clientcmd.ClientConfig
	Offline              bool
	OfflineFiles         []string
	Context              context.Context
}

func (o Opts) String() string {
	return fmt.Sprintf("(argoServerOpts=%v,instanceID=%v)", o.ArgoServerOpts, o.InstanceID)
}

func (o *Opts) GetContext() context.Context {
	if o.Context == nil {
		o.Context = logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	}

	return o.Context
}

// DEPRECATED: use NewClientFromOpts
func NewClient(argoServer string, authSupplier func() string, clientConfig clientcmd.ClientConfig) (context.Context, Client, error) {
	return NewClientFromOpts(Opts{
		ArgoServerOpts: ArgoServerOpts{URL: argoServer},
		AuthSupplier:   authSupplier,
		ClientConfigSupplier: func() clientcmd.ClientConfig {
			return clientConfig
		},
		Context: logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())),
	})
}

func NewClientFromOpts(opts Opts) (context.Context, Client, error) {
	ctx := opts.GetContext()
	if ctx == nil {
		panic("ctx was nil mate")
	}
	log := logging.GetLoggerFromContext(ctx)
	if log == nil {
		log = logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
		ctx = logging.WithLogger(ctx, log)
	}
	log.WithField("opts", opts).Debug(ctx, "Client options")
	if opts.Offline {
		return newOfflineClient(opts.OfflineFiles)
	}
	if opts.ArgoServerOpts.URL != "" && opts.InstanceID != "" {
		return nil, nil, fmt.Errorf("cannot use instance ID with Argo Server")
	}
	if opts.ArgoServerOpts.HTTP1 {
		if opts.AuthSupplier == nil {
			return nil, nil, fmt.Errorf("AuthSupplier cannot be empty when connecting to Argo Server")
		}
		return newHTTP1Client(opts.ArgoServerOpts.GetURL(), opts.AuthSupplier(), opts.ArgoServerOpts.InsecureSkipVerify, opts.ArgoServerOpts.Headers, opts.ArgoServerOpts.HTTP1Client)
	} else if opts.ArgoServerOpts.URL != "" {
		if opts.AuthSupplier == nil {
			return nil, nil, fmt.Errorf("AuthSupplier cannot be empty when connecting to Argo Server")
		}
		return newArgoServerClient(opts.ArgoServerOpts, opts.AuthSupplier())
	} else {
		if opts.ClientConfigSupplier != nil {
			opts.ClientConfig = opts.ClientConfigSupplier()
		}

		ctx, client, err := newArgoKubeClient(opts.GetContext(), opts.ArgoKubeOpts, opts.ClientConfig, instanceid.NewService(opts.InstanceID))
		return ctx, client, err
	}
}
