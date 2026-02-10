package apiclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type Client interface {
	NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error)
	NewWorkflowServiceClient(ctx context.Context) workflowpkg.WorkflowServiceClient
	NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error)
	NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error)
	NewClusterWorkflowTemplateServiceClient() (clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient, error)
	NewInfoServiceClient() (infopkg.InfoServiceClient, error)
	NewSyncServiceClient(ctx context.Context) (syncpkg.SyncServiceClient, error)
}

type Opts struct {
	ArgoServerOpts ArgoServerOpts
	ArgoKubeOpts   ArgoKubeOpts
	InstanceID     string
	AuthSupplier   func() string
	// Deprecated: use ClientConfigSupplier
	ClientConfig         clientcmd.ClientConfig
	ClientConfigSupplier func() clientcmd.ClientConfig
	Offline              bool
	OfflineFiles         []string
	// Deprecated: use NewClientFromOptsWithContext
	//nolint: containedctx
	Context   context.Context
	LogLevel  string
	LogFormat string
}

func (o Opts) String() string {
	return fmt.Sprintf("(argoServerOpts=%v,instanceID=%v)", o.ArgoServerOpts, o.InstanceID)
}

func NewClientFromOptsWithContext(ctx context.Context, opts Opts) (context.Context, Client, error) {
	log := logging.GetLoggerFromContextOrNil(ctx)
	if log == nil {
		logLevel, err := logging.ParseLevelOr(opts.LogLevel, logging.Info)
		if err != nil {
			return nil, nil, err
		}
		logFormat, err := logging.TypeFromStringOr(opts.LogFormat, logging.Text)
		if err != nil {
			return nil, nil, err
		}
		log = logging.NewSlogLogger(logLevel, logFormat)
		ctx = logging.WithLogger(ctx, log)
	}
	log.WithField("opts", opts).Debug(ctx, "Client options")
	if opts.Offline {
		return newOfflineClient(ctx, opts.OfflineFiles)
	}
	if opts.ArgoServerOpts.URL != "" && opts.InstanceID != "" {
		return nil, nil, fmt.Errorf("cannot use instance ID with Argo Server")
	}
	switch {
	case opts.ArgoServerOpts.HTTP1:
		if opts.AuthSupplier == nil {
			return nil, nil, fmt.Errorf("AuthSupplier cannot be empty when connecting to Argo Server")
		}
		return newHTTP1Client(ctx, opts.ArgoServerOpts.GetURL(), opts.AuthSupplier(), opts.ArgoServerOpts.InsecureSkipVerify, opts.ArgoServerOpts.Headers, opts.ArgoServerOpts.HTTP1Client)
	case opts.ArgoServerOpts.URL != "":
		if opts.AuthSupplier == nil {
			return nil, nil, fmt.Errorf("AuthSupplier cannot be empty when connecting to Argo Server")
		}
		return newArgoServerClient(ctx, opts.ArgoServerOpts, opts.AuthSupplier())
	default:
		if opts.ClientConfigSupplier != nil {
			opts.ClientConfig = opts.ClientConfigSupplier()
		}
		return newArgoKubeClient(ctx, opts.ArgoKubeOpts, opts.ClientConfig, instanceid.NewService(opts.InstanceID))
	}
}
