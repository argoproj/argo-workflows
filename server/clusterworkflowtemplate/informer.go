package clusterworkflowtemplate

import (
	"context"
	"os"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/informer"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

const (
	workflowTemplateResyncPeriod = 20 * time.Minute
)

type Informer interface {
	Run(ctx context.Context, stopCh <-chan struct{})
	Getter(ctx context.Context) templateresolution.ClusterWorkflowTemplateGetter
}

var _ types.ClusterWorkflowTemplateStore = &informerImpl{}

type informerImpl struct {
	informer wfextvv1alpha1.ClusterWorkflowTemplateInformer
}

func NewInformer(restConfig *rest.Config) (Informer, error) {
	dynamicInterface, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	informer := informer.NewTolerantClusterWorkflowTemplateInformer(
		dynamicInterface,
		workflowTemplateResyncPeriod,
	)
	return &informerImpl{
		informer: informer,
	}, nil
}

// Start informer in separate go-routine and block until cache sync
func (cwti *informerImpl) Run(ctx context.Context, stopCh <-chan struct{}) {
	go cwti.informer.Informer().Run(stopCh)

	if !cache.WaitForCacheSync(
		stopCh,
		cwti.informer.Informer().HasSynced,
	) {
		logging.RequireLoggerFromContext(ctx).WithFatal().Error(ctx, "Timed out waiting for caches to sync")
		os.Exit(1)
	}
}

// if namespace contains empty string Lister will use the namespace provided during initialization
func (cwti *informerImpl) Getter(ctx context.Context) templateresolution.ClusterWorkflowTemplateGetter {
	if cwti == nil || cwti.informer == nil {
		logging.RequireLoggerFromContext(ctx).Error(ctx, "Template informer not started")
		return nil
	}
	return templateresolution.WrapClusterWorkflowTemplateLister(cwti.informer.Lister())
}
