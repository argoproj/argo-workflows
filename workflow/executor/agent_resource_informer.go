package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// ResourceHandler is invoked on every event for any monitored resource. The
// deleted flag distinguishes a normal add/update from a delete event. The
// handler is responsible for routing — e.g., by reading a nodeID label off
// the object's metadata — since a single handler covers every monitored
// object across every GVR.
type ResourceHandler func(ctx context.Context, obj *unstructured.Unstructured, deleted bool)

// MonitoredResourceInformer manages a set of dynamic informers — one per
// GroupVersionResource — each filtered to objects carrying the
// common.LabelKeyMonitoredResource=<workflowName> label. The agent uses it
// to observe k8s objects created by resource templates and react to their
// state changes without spawning a wait pod per resource.
//
// The label selector already scopes the watch to objects this workflow
// created, so a single handler (registered once per GVR informer at
// creation time) fires for every relevant event. Callers tell the
// informer which GVRs to watch via Watch, which is idempotent.
type MonitoredResourceInformer struct {
	dynClient    dynamic.Interface
	namespace    string
	workflowName string
	resyncPeriod time.Duration
	handler      ResourceHandler

	mu        sync.Mutex
	informers map[schema.GroupVersionResource]*gvrInformer
}

type gvrInformer struct {
	informer cache.SharedIndexInformer
	stop     chan struct{}
}

// NewMonitoredResourceInformer constructs an informer manager. If namespace
// is empty the informers run cluster-wide (subject to RBAC); otherwise they
// are scoped to that namespace.
func NewMonitoredResourceInformer(dynClient dynamic.Interface, namespace, workflowName string, resyncPeriod time.Duration, handler ResourceHandler) *MonitoredResourceInformer {
	return &MonitoredResourceInformer{
		dynClient:    dynClient,
		namespace:    namespace,
		workflowName: workflowName,
		resyncPeriod: resyncPeriod,
		handler:      handler,
		informers:    map[schema.GroupVersionResource]*gvrInformer{},
	}
}

func (m *MonitoredResourceInformer) labelSelector() string {
	return fmt.Sprintf("%s=%s", common.LabelKeyMonitoredResource, m.workflowName)
}

// Watch ensures a dynamic informer is running for the given GVR, scoped to
// the configured namespace and filtered by the monitored-resource label
// selector. Idempotent — repeated calls for the same GVR are a no-op after
// the first.
//
// Blocks until the informer's cache has synced or ctx is cancelled.
func (m *MonitoredResourceInformer) Watch(ctx context.Context, gvr schema.GroupVersionResource) error {
	logger := logging.RequireLoggerFromContext(ctx)

	m.mu.Lock()
	inf, exists := m.informers[gvr]
	if !exists {
		factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
			m.dynClient,
			m.resyncPeriod,
			m.namespace,
			func(opts *metav1.ListOptions) {
				opts.LabelSelector = m.labelSelector()
			},
		)
		shared := factory.ForResource(gvr).Informer()
		if _, err := shared.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj any) { m.dispatch(ctx, obj, false) },
			UpdateFunc: func(_, obj any) { m.dispatch(ctx, obj, false) },
			DeleteFunc: func(obj any) { m.dispatch(ctx, obj, true) },
		}); err != nil {
			m.mu.Unlock()
			return fmt.Errorf("register dispatcher for %s: %w", gvr, err)
		}
		stop := make(chan struct{})
		inf = &gvrInformer{informer: shared, stop: stop}
		m.informers[gvr] = inf
		go shared.Run(stop)
		logger.
			WithField("gvr", gvr.String()).
			WithField("namespace", m.namespace).
			WithField("selector", m.labelSelector()).
			Info(ctx, "Started monitored-resource informer")
	}
	m.mu.Unlock()

	if !cache.WaitForCacheSync(ctx.Done(), inf.informer.HasSynced) {
		return fmt.Errorf("cache sync failed for %s", gvr)
	}
	return nil
}

// Get returns the cached object for gvr/namespace/name. The boolean
// indicates presence; an error is returned if no informer is running for
// gvr or the store lookup fails.
func (m *MonitoredResourceInformer) Get(gvr schema.GroupVersionResource, namespace, name string) (any, bool, error) {
	m.mu.Lock()
	inf, ok := m.informers[gvr]
	m.mu.Unlock()
	if !ok {
		return nil, false, fmt.Errorf("no informer registered for %s", gvr)
	}
	key := name
	if namespace != "" {
		key = namespace + "/" + name
	}
	return inf.informer.GetStore().GetByKey(key)
}

// Stop stops every informer this MonitoredResourceInformer is managing.
func (m *MonitoredResourceInformer) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for gvr, inf := range m.informers {
		close(inf.stop)
		delete(m.informers, gvr)
	}
}

func (m *MonitoredResourceInformer) dispatch(ctx context.Context, obj any, deleted bool) {
	u, ok := toUnstructured(obj)
	if !ok {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx).
		WithField("namespace", u.GetNamespace()).
		WithField("name", u.GetName()).
		WithField("gvk", u.GetObjectKind().GroupVersionKind().String())
	if wf, err := wfutil.FromUnstructured(u); err != nil {
		logger = logger.WithField("phaseConvertErr", err.Error())
	} else {
		logger = logger.WithField("phase", string(wf.Status.Phase))
	}
	logger.Info(ctx, "received an event")

	if m.handler != nil {
		m.handler(ctx, u, deleted)
	} else {
		logger.Info(ctx, "handler wasn't set")
	}
}

func toUnstructured(obj any) (*unstructured.Unstructured, bool) {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, true
	}
	if tomb, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if u, ok := tomb.Obj.(*unstructured.Unstructured); ok {
			return u, true
		}
	}
	return nil, false
}
