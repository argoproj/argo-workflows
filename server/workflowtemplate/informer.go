package workflowtemplate

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/informer"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

const (
	workflowTemplateResyncPeriod = 20 * time.Minute
)

var _ types.WorkflowTemplateStore = &Informer{}

type Informer struct {
	managedNamespace string
	informer         wfextvv1alpha1.WorkflowTemplateInformer
}

func NewInformer(restConfig *rest.Config, managedNamespace string) (*Informer, error) {
	dynamicInterface, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	informer := informer.NewTolerantWorkflowTemplateInformer(
		dynamicInterface,
		workflowTemplateResyncPeriod,
		managedNamespace)
	return &Informer{
		informer:         informer,
		managedNamespace: managedNamespace,
	}, nil
}

// Start informer in separate go-routine and block until cache sync
func (wti *Informer) Run(stopCh <-chan struct{}) {

	wti.informer.Informer()

	go wti.informer.Informer().Run(stopCh)

	if !cache.WaitForCacheSync(
		stopCh,
		wti.informer.Informer().HasSynced,
	) {
		log.Fatal("Timed out waiting for caches to sync")
	}
}

// if namespace contains empty string Lister will use the namespace provided during initialization
func (wti *Informer) Getter(_ context.Context, namespace string) templateresolution.WorkflowTemplateNamespacedGetter {
	if wti.informer == nil {
		log.Fatal("Template informer not started")
	}
	if namespace == "" {
		namespace = wti.managedNamespace
	}
	return wti.informer.Lister().WorkflowTemplates(namespace)
}
