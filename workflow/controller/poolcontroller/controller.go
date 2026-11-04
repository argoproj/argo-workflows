// Package poolcontroller runs hot worker pools. It creates a pool's
// Deployment once a pooled WorkflowTaskSet of the pool exists, deletes it
// once the pool has had none for its retention, and reports each pool's
// health for the nodes waiting on it (dispatch.PoolHealth).
//
// A pool is identified by its namespace and hash, the pool-hash label of its
// TaskSets, and served by one Deployment built from a TaskSet's spec.pool
// alone. Once created, a Deployment's spec is never changed: its replicas
// belong to whatever scales the pool, and its hash fixes the rest. Only the
// retention annotation is kept up to date. The controller touches only the
// Deployments of this controller instance's pools, and never one whose labels
// name another pool, even if it has the pool's Deployment name. It builds and
// changes only the pools whose spec.pool its own version wrote, since its pod
// template may not suit another version's executor image: another version's
// pool is left to its retention, and the operator enqueues that pool's
// unclaimed tasks again for this version.
package poolcontroller

import (
	"context"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/events"
)

// DeploymentVerbs are the verbs on apps Deployments the controller uses, which
// the optional hot workers RBAC component grants.
var DeploymentVerbs = []string{"get", "list", "watch", "create", "patch", "delete"}

// taskSetBatch is how long a pool whose Deployment exists waits after a
// TaskSet of it comes or goes before it is reconciled, so a busy pool is
// reconciled, and its newest TaskSet read, about once per batch rather than
// once per task.
const taskSetBatch = time.Second

// Builder builds the Deployment of the pool with hash in namespace from its
// spec.pool, or fails if this controller can't build that spec.
type Builder func(namespace, hash string, spec *wfv1.PoolSpec) (*appsv1.Deployment, error)

// Options are what a Controller works from. The informers must be indexed by
// indexes.PoolIndex and select this controller instance's objects only.
type Options struct {
	KubeClient kubernetes.Interface
	WfClient   wfclientset.Interface
	// TaskSets caches the metadata of pooled WorkflowTaskSets.
	TaskSets cache.SharedIndexInformer
	// Pods caches pool pods.
	Pods cache.SharedIndexInformer
	// Deployments caches pools' Deployments.
	Deployments cache.SharedIndexInformer
	// Queue holds the keys of the pools to reconcile; NewRateLimiter suits it.
	Queue      workqueue.TypedRateLimitingInterface[string]
	InstanceID string
	// ControllerVersion is this controller's version, which a pool's
	// spec.pool must carry for the controller to build or change the pool.
	ControllerVersion string
	Build             Builder
	// DefaultRetention is the retention of a pool whose Deployment doesn't
	// record one.
	DefaultRetention func() time.Duration
	Events           events.EventRecorderManager
	// OnHealthChange is called, with no lock held, after a pool's health
	// changes. A pool the controller knows nothing about counts as OK.
	OnHealthChange func(namespace, hash string)
}

// Controller reconciles pools. It implements dispatch.PoolHealth.
type Controller struct {
	Options
	now func() time.Time

	mu    sync.Mutex
	pools map[string]*pool
}

var _ dispatch.PoolHealth = &Controller{}

// pool is what the controller remembers of a pool, by key. Only the worker
// reconciling the pool touches it, apart from health, which mu guards.
type pool struct {
	// idleSince is when the pool was first seen without TaskSets, zero while
	// it has some. It is kept in memory only, so a controller restart restarts
	// the retention timer rather than shortening it.
	idleSince time.Time
	// spec is the spec.pool of the TaskSet with UID specFrom, read live, as
	// the TaskSet cache holds metadata only. It may be nil.
	specFrom types.UID
	spec     *wfv1.PoolSpec
	// taskSet refers to that TaskSet, for events.
	taskSet *apiv1.ObjectReference
	health  dispatch.Health
}

// NewRateLimiter is the rate limiter for a Controller's queue: it backs off
// a pool that keeps failing, such as one whose Deployment can't be created,
// up to five minutes between attempts.
func NewRateLimiter() workqueue.TypedRateLimiter[string] {
	return workqueue.NewTypedItemExponentialFailureRateLimiter[string](time.Second, 5*time.Minute)
}

// New returns a Controller, with its handlers added to the informers.
func New(opts Options) *Controller {
	c := &Controller{Options: opts, now: time.Now, pools: map[string]*pool{}}
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started (https://github.com/kubernetes/client-go/blob/46588f2726fa3e25b1704d6418190f424f95a990/tools/cache/shared_informer.go#L580)
	opts.TaskSets.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// A pooled TaskSet is written once and never updated.
		AddFunc:    c.enqueueAfterTaskSet,
		DeleteFunc: c.enqueueAfterTaskSet,
	})
	//nolint:errcheck // as above
	opts.Pods.AddEventHandler(changeHandlers(c.enqueue))
	//nolint:errcheck // as above
	opts.Deployments.AddEventHandler(changeHandlers(c.enqueue))
	return c
}

// changeHandlers call f with every added, changed or deleted object.
func changeHandlers(f func(any)) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: f,
		UpdateFunc: func(old, newObj any) {
			if informerutil.ResourceVersionUnchanged(old, newObj) {
				return
			}
			f(newObj)
		},
		DeleteFunc: f,
	}
}

// keyOf is the key of the pool an object belongs to: its namespace and pool
// hash, as indexes.PoolIndex has it.
func keyOf(obj any) (string, bool) {
	m, err := meta.Accessor(informerutil.Untombstone(obj))
	if err != nil {
		return "", false
	}
	hash := m.GetLabels()[common.LabelKeyPoolHash]
	if hash == "" {
		return "", false
	}
	return indexes.MetaNamespaceLabelIndex(m.GetNamespace(), hash), true
}

func (c *Controller) enqueue(obj any) {
	if key, ok := keyOf(obj); ok {
		c.Queue.Add(key)
	}
}

// enqueueAfterTaskSet queues a pool after a TaskSet of it comes or goes: at
// once if the pool has no Deployment yet, which a cold start waits for, and
// otherwise after taskSetBatch.
func (c *Controller) enqueueAfterTaskSet(obj any) {
	key, ok := keyOf(obj)
	if !ok {
		return
	}
	if deployments, err := c.Deployments.GetIndexer().ByIndex(indexes.PoolIndex, key); err == nil && len(deployments) > 0 {
		c.Queue.AddAfter(key, taskSetBatch)
		return
	}
	c.Queue.Add(key)
}

// Health implements dispatch.PoolHealth.
func (c *Controller) Health(namespace, hash string) (dispatch.Health, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.pools[indexes.MetaNamespaceLabelIndex(namespace, hash)]
	if !ok {
		return dispatch.Health{}, false
	}
	return p.health, true
}

// Run reconciles pools with the workers, once the informers have synced,
// until ctx is done.
func (c *Controller) Run(ctx context.Context, workers int) {
	defer c.Queue.ShutDown()
	ctx, logger := logging.RequireLoggerFromContext(ctx).WithField("component", "pool_controller").InContext(ctx)
	if !cache.WaitForCacheSync(ctx.Done(), c.TaskSets.HasSynced, c.Pods.HasSynced, c.Deployments.HasSynced) {
		return
	}
	logger.Info(ctx, "Pool controller started")
	for range workers {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}
	<-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	after, err := c.reconcile(ctx, key)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithField("pool", key).WithError(err).Warn(ctx, "Failed to reconcile pool")
		c.Queue.AddRateLimited(key)
		return true
	}
	c.Queue.Forget(key)
	if after > 0 {
		c.Queue.AddAfter(key, after)
	}
	return true
}

// state returns the pool's state, remembering a new pool as OK.
func (c *Controller) state(key string) *pool {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.pools[key]
	if !ok {
		p = &pool{health: dispatch.Health{State: dispatch.HealthOK}}
		c.pools[key] = p
	}
	return p
}

// forget forgets a pool that has neither TaskSets nor a Deployment.
func (c *Controller) forget(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pools, key)
}

func splitKey(key string) (namespace, hash string) {
	namespace, hash, _ = strings.Cut(key, "/")
	return namespace, hash
}
