package informer

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	extwfv1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTaskSetInformer struct {
	delegate informers.GenericInformer
}

// a drop-in replacement for `extwfv1.WorkflowTemplateInformer` that ignores malformed resources
func NewTolerantWorkflowTaskSetInformer(dynamicInterface dynamic.Interface, defaultResync time.Duration, namespace string) extwfv1.WorkflowTaskSetInformer {
	return &tolerantWorkflowTaskSetInformer{delegate: dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicInterface, defaultResync, namespace, func(options *metav1.ListOptions) {}).
		ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowTaskSetPlural})}
}

func (t *tolerantWorkflowTaskSetInformer) Informer() cache.SharedIndexInformer {
	return t.delegate.Informer()
}

func (t *tolerantWorkflowTaskSetInformer) Lister() v1alpha1.WorkflowTaskSetLister {
	return &tolerantWorkflowTaskSetLister{delegate: t.delegate.Lister()}
}
