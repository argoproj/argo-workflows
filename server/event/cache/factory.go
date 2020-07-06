package cache

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// the goal of this controller is to be very low in memory usage by only storing the `key` objects that are already
// known about
func NewFilterUsingKeyController(restClient rest.Interface, namespace string, req labels.Selector, resource string, objectType runtime.Object, filterFunc func(d cache.Delta) bool) (cache.Controller, cache.KeyLister) {
	knownObjects := newStore()
	return cache.New(&cache.Config{
		Queue: cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, knownObjects),
		ListerWatcher: &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = req.String()
				return restClient.Get().
					Namespace(namespace).
					Resource(resource).
					VersionedParams(&options, metav1.ParameterCodec).
					Do().
					Get()
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = req.String()
				options.Watch = true
				return restClient.Get().
					Namespace(namespace).
					Resource(resource).
					VersionedParams(&options, metav1.ParameterCodec).
					Watch()
			},
		},
		ObjectType: objectType,
		// note - we never re-sync
		Process: func(obj interface{}) error {
			for _, d := range obj.(cache.Deltas) {
				switch d.Type {
				case cache.Added, cache.Updated:
					if filterFunc(d) {
						knownObjects.Add(d.Object)
					} else {
						knownObjects.Delete(d.Object)
					}
				case cache.Deleted:
					knownObjects.Delete(d.Object)
				}
			}
			return nil
		},
	}), knownObjects
}
