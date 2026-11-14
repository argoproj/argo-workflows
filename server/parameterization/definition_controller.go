package parameterization

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// DefinitionController owns only definition status subresources. Its service
// account must have no Workflow, spec, receipt, Secret or provider write rights.
type DefinitionController struct {
	ready     atomic.Bool
	client    dynamic.Interface
	namespace string
	factories []dynamicinformer.DynamicSharedInformerFactory
	informers map[string]cache.SharedIndexInformer
	queue     workqueue.TypedRateLimitingInterface[DefinitionKey]
}

func (c *DefinitionController) Ready() bool { return c.ready.Load() }

func definitionGVR(resource string) schema.GroupVersionResource {
	gvr := schema.GroupVersionResource{Group: p.Group, Version: p.Version, Resource: resource}
	if resource == "workflowtemplates" || resource == "clusterworkflowtemplates" {
		gvr.Group = "argoproj.io"
		gvr.Version = "v1alpha1"
	}
	return gvr
}

func definitionIndex(key DefinitionKey) string { return key.Resource + "/" + key.Name }

func NewDefinitionController(client dynamic.Interface, namespace string, clusterTemplates bool) (*DefinitionController, error) {
	if namespace == "" {
		return nil, fmt.Errorf("a single namespace is required")
	}
	namespaced := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, namespace, nil)
	c := &DefinitionController{client: client, namespace: namespace, factories: []dynamicinformer.DynamicSharedInformerFactory{namespaced}, informers: map[string]cache.SharedIndexInformer{}, queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[DefinitionKey]())}
	resources := []string{"parametertemplates", "workflowforms", "workflowparameterbindings", "actiondefinitions", "actionresulttemplates", "workflowactionbindings", "workflowtemplates"}
	if clusterTemplates {
		resources = append(resources, "clusterworkflowtemplates")
	}
	for _, resource := range resources {
		factory := namespaced
		if resource == "clusterworkflowtemplates" {
			factory = dynamicinformer.NewDynamicSharedInformerFactory(client, 0)
			c.factories = append(c.factories, factory)
		}
		informer := factory.ForResource(definitionGVR(resource)).Informer()
		if err := informer.AddIndexers(cache.Indexers{"definitionReferences": func(obj any) ([]string, error) {
			o, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return nil, nil
			}
			var result []string
			for _, key := range definitionDependencies(o) {
				result = append(result, definitionIndex(key))
			}
			return result, nil
		}}); err != nil {
			return nil, err
		}
		c.informers[resource] = informer
		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) { c.changed(resource, obj) },
			UpdateFunc: func(oldObj, newObj any) {
				old, next := oldObj.(*unstructured.Unstructured), newObj.(*unstructured.Unstructured)
				// Status updates must not cause reference-resourceVersion feedback.
				if old.GetUID() != next.GetUID() || old.GetGeneration() != next.GetGeneration() || !reflect.DeepEqual(old.Object["spec"], next.Object["spec"]) || !reflect.DeepEqual(old.GetDeletionTimestamp(), next.GetDeletionTimestamp()) {
					c.changed(resource, newObj)
				}
			},
			DeleteFunc: func(obj any) { c.changed(resource, obj) },
		})
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *DefinitionController) changed(resource string, obj any) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	o, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	pending := []DefinitionKey{{resource, o.GetName()}}
	seen := map[DefinitionKey]bool{}
	for len(pending) > 0 {
		key := pending[0]
		pending = pending[1:]
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, editable := editableDefinitions[key.Resource]; editable {
			c.queue.Add(key)
		}
		for r, informer := range c.informers {
			objects, err := informer.GetIndexer().ByIndex("definitionReferences", definitionIndex(key))
			if err != nil {
				continue
			}
			for _, dependent := range objects {
				pending = append(pending, DefinitionKey{r, dependent.(*unstructured.Unstructured).GetName()})
			}
		}
	}
}

func (c *DefinitionController) lookup(key DefinitionKey) (*unstructured.Unstructured, error) {
	informer := c.informers[key.Resource]
	if informer == nil {
		return nil, fmt.Errorf("reference scope is disabled")
	}
	name := c.namespace + "/" + key.Name
	if key.Resource == "clusterworkflowtemplates" {
		name = key.Name
	}
	x, exists, err := informer.GetStore().GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apierrors.NewNotFound(definitionGVR(key.Resource).GroupResource(), key.Name)
	}
	return x.(*unstructured.Unstructured), nil
}

func stableDefinitionStatus(current *unstructured.Unstructured, next parameterapi.DefinitionStatus) (map[string]any, error) {
	var old parameterapi.DefinitionStatus
	if data, ok := current.Object["status"].(map[string]any); ok {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(data, &old); err != nil {
			return nil, err
		}
	}
	// Preserve the original observation RV when the referenced UID/spec is
	// unchanged. Kubernetes status writes advance RV but not semantic versions.
	for i, ref := range next.ResolvedReferences {
		for _, previous := range old.ResolvedReferences {
			if ref.Kind == previous.Kind && ref.Name == previous.Name && ref.UID == previous.UID && ref.Digest == previous.Digest {
				next.ResolvedReferences[i].ResourceVersion = previous.ResourceVersion
				break
			}
		}
	}
	conditions := old.Conditions
	for _, condition := range next.Conditions {
		meta.SetStatusCondition(&conditions, condition)
	}
	next.Conditions = conditions
	return runtime.DefaultUnstructuredConverter.ToUnstructured(&next)
}

func (c *DefinitionController) reconcile(ctx context.Context, key DefinitionKey) error {
	if _, ok := editableDefinitions[key.Resource]; !ok {
		return nil
	}
	current, err := c.lookup(key)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if current.GetDeletionTimestamp() != nil {
		return nil
	}
	status, err := stableDefinitionStatus(current, EvaluateDefinition(current, c.lookup))
	if err != nil {
		return err
	}
	if reflect.DeepEqual(current.Object["status"], status) {
		return nil
	}
	next := current.DeepCopy()
	next.Object["status"] = status
	_, err = c.client.Resource(definitionGVR(key.Resource)).Namespace(c.namespace).UpdateStatus(ctx, next, metav1.UpdateOptions{})
	return err
}

// Run uses informers and a reverse-reference index. No request scans the full
// catalog. Conflicts requeue the affected definition; no spec is ever written.
func (c *DefinitionController) Run(ctx context.Context, onError func(error)) error {
	defer c.queue.ShutDown()
	defer c.ready.Store(false)
	for _, factory := range c.factories {
		factory.Start(ctx.Done())
	}
	for _, informer := range c.informers {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			return fmt.Errorf("definition cache did not synchronize")
		}
	}
	c.ready.Store(true)
	go func() { <-ctx.Done(); c.queue.ShutDown() }()
	for {
		key, shutdown := c.queue.Get()
		if shutdown {
			return nil
		}
		attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := c.reconcile(attempt, key)
		cancel()
		if err != nil && ctx.Err() == nil {
			if onError != nil {
				onError(fmt.Errorf("definition %s: %w", definitionIndex(key), err))
			}
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}
		c.queue.Done(key)
	}
}
