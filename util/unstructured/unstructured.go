package unstructured

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/tools/cache"
)

const workflowPaginationLimit = 500

// NewUnstructuredInformer constructs a new informer for Unstructured type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewUnstructuredInformer(ctx context.Context, resource schema.GroupVersionResource, client dynamic.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredUnstructuredInformer(ctx, resource, client, namespace, resyncPeriod, indexers, nil, nil)
}

// NewFilteredUnstructuredInformer constructs a new informer for Unstructured type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredUnstructuredInformer(ctx context.Context, resource schema.GroupVersionResource, client dynamic.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListRequestListOptions internalinterfaces.TweakListOptionsFunc, tweakWatchRequestListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListRequestListOptions != nil {
					tweakListRequestListOptions(&options)
				}
				var allWorkflows []unstructured.Unstructured
				continueTok := ""
				options.Limit = workflowPaginationLimit
				for {
					options.Continue = continueTok
					unList, err := client.Resource(resource).Namespace(namespace).List(ctx, options)
					if err != nil {
						return nil, err
					}
					allWorkflows = append(allWorkflows, unList.Items...)

					if unList.GetContinue() == "" {
						break
					}
					continueTok = unList.GetContinue()
				}
				return &unstructured.UnstructuredList{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "List",
					},
					Items: allWorkflows,
				}, nil
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakWatchRequestListOptions != nil {
					tweakWatchRequestListOptions(&options)
				}
				return client.Resource(resource).Namespace(namespace).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		resyncPeriod,
		indexers,
	)
}
