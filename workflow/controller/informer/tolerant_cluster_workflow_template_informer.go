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
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type tolerantClusterWorkflowTemplateInformer struct {
	delegate informers.GenericInformer
}

// a drop-in replacement for `extwfv1.ClusterWorkflowTemplateInformer` that ignores malformed resources
func NewTolerantClusterWorkflowTemplateInformer(dynamicInterface dynamic.Interface, defaultResync time.Duration) extwfv1.ClusterWorkflowTemplateInformer {
	return &tolerantClusterWorkflowTemplateInformer{delegate: dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicInterface, defaultResync, "", func(options *metav1.ListOptions) {
		// `ResourceVersion=0` does not honor the `limit` in API calls, which results in making significant List calls
		// without `limit`. For details, see https://github.com/argoproj/argo-workflows/pull/11343
		// The reflector will record `lastSyncResourceVersion`. If `ResourceVersion != "0"`, we should use the value
		// recorded by the reflector. see https://github.com/argoproj/argo-workflows/pull/13466
		if options.ResourceVersion == "0" {
			options.ResourceVersion = ""
		}
		// The reflector will set the Limit to `0` when `ResourceVersion != "" && ResourceVersion != "0"`, which will fail
		// to limit the number of workflow returns. Timeouts and other errors may occur when there are a lots of workflows.
		// see https://github.com/kubernetes/client-go/blob/ee1a5aaf793a9ace9c433f5fb26a19058ed5f37c/tools/cache/reflector.go#L286
		if options.Limit == 0 {
			options.Limit = common.DefaultPageSize
		}
	}).ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.ClusterWorkflowTemplatePlural})}
}

func (t *tolerantClusterWorkflowTemplateInformer) Informer() cache.SharedIndexInformer {
	return t.delegate.Informer()
}

func (t *tolerantClusterWorkflowTemplateInformer) Lister() v1alpha1.ClusterWorkflowTemplateLister {
	return &tolerantClusterWorkflowTemplateLister{delegate: t.delegate.Lister()}
}
