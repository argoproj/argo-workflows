package informer

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	extwfv1 "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/listers/workflow/v1alpha1"
)

type tolerantClusterWorkflowTemplateInformer struct {
	delegate informers.GenericInformer
}

// NewTolerantClusterWorkflowTemplateInformer is a drop-in replacement for `extwfv1.ClusterWorkflowTemplateInformer` that ignores malformed resources.
func NewTolerantClusterWorkflowTemplateInformer(dynamicInterface dynamic.Interface, defaultResync time.Duration) extwfv1.ClusterWorkflowTemplateInformer {
	return &tolerantClusterWorkflowTemplateInformer{delegate: dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicInterface, defaultResync, "", func(options *metav1.ListOptions) {
		// `ResourceVersion=0` does not honor the `limit` in API calls, which results in making significant List calls
		// without `limit`. For details, see https://github.com/argoproj/argo-workflows/pull/11343
		// Check if ResourceVersion is "0" and reset it to empty string to ensure proper pagination behavior
		if options.ResourceVersion == "0" {
			options.ResourceVersion = ""
		}
	}).ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.ClusterWorkflowTemplatePlural})}
}

func (t *tolerantClusterWorkflowTemplateInformer) Informer() cache.SharedIndexInformer {
	return t.delegate.Informer()
}

func (t *tolerantClusterWorkflowTemplateInformer) Lister() v1alpha1.ClusterWorkflowTemplateLister {
	return &tolerantClusterWorkflowTemplateLister{delegate: t.delegate.Lister()}
}
