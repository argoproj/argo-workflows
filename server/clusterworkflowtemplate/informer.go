package clusterworkflowtemplate

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	clientv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/informer"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

const (
	workflowTemplateResyncPeriod = 20 * time.Minute
)

var _ types.ClusterWorkflowTemplateStore = &Informer{}

type Informer struct {
	informer wfextvv1alpha1.ClusterWorkflowTemplateInformer
}

func NewInformer(restConfig *rest.Config) (*Informer, error) {
	dynamicInterface, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	informer := informer.NewTolerantClusterWorkflowTemplateInformer(
		dynamicInterface,
		workflowTemplateResyncPeriod,
	)
	return &Informer{
		informer: informer,
	}, nil
}

// Start informer in separate go-routine and block until cache sync
func (cwti *Informer) Run(stopCh <-chan struct{}) {

	cwti.informer.Informer()

	go cwti.informer.Informer().Run(stopCh)

	if !cache.WaitForCacheSync(
		stopCh,
		cwti.informer.Informer().HasSynced,
	) {
		log.Fatal("Timed out waiting for caches to sync")
	}
}

// if namespace contains empty string Lister will use the namespace provided during initialization
func (cwti *Informer) Lister(_ context.Context, namespace string) clientv1alpha1.ClusterWorkflowTemplateLister {
	if cwti.informer == nil {
		log.Fatal("Template informer not started")
	}
	return cwti.informer.Lister()
}

// if namespace contains empty string Lister will use the namespace provided during initialization
func (cwti *Informer) Getter(_ context.Context) templateresolution.ClusterWorkflowTemplateGetter {
	if cwti.informer == nil {
		log.Fatal("Template informer not started")
	}
	return cwti.informer.Lister()
}
