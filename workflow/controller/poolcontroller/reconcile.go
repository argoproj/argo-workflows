package poolcontroller

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// reconcile brings the pool with key towards its desired state: a Deployment
// while it has TaskSets or is within its retention, none otherwise. It
// returns how long to wait before reconciling the pool again, if nothing
// else requeues it first.
func (c *Controller) reconcile(ctx context.Context, key string) (time.Duration, error) {
	namespace, hash := splitKey(key)
	ctx, _ = logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"namespace": namespace, "poolHash": hash}).InContext(ctx)
	taskSets, err := c.TaskSets.GetIndexer().ByIndex(indexes.PoolIndex, key)
	if err != nil {
		return 0, err
	}
	deploy, err := c.cachedDeployment(key)
	if err != nil {
		return 0, err
	}
	if len(taskSets) == 0 {
		return c.reconcileIdle(ctx, key, deploy)
	}
	return c.reconcileBusy(ctx, key, taskSets, deploy)
}

// cachedDeployment is the pool's Deployment in the cache, or nil.
func (c *Controller) cachedDeployment(key string) (*appsv1.Deployment, error) {
	objs, err := c.Deployments.GetIndexer().ByIndex(indexes.PoolIndex, key)
	if err != nil || len(objs) == 0 {
		return nil, err
	}
	// Only a Deployment someone else labelled with the pool's hash makes two.
	slices.SortFunc(objs, func(a, b any) int {
		return cmp.Compare(a.(*appsv1.Deployment).Name, b.(*appsv1.Deployment).Name)
	})
	return objs[0].(*appsv1.Deployment), nil
}

// reconcileIdle handles a pool with no TaskSets: it deletes its Deployment
// once the pool has been idle for its retention, and forgets a pool with no
// Deployment.
func (c *Controller) reconcileIdle(ctx context.Context, key string, deploy *appsv1.Deployment) (time.Duration, error) {
	if deploy == nil {
		c.forget(key)
		return 0, nil
	}
	p := c.state(key)
	c.setHealth(ctx, key, podHealth(deploy, c.cachedPods(key)), deploymentRef(deploy))
	if deploy.DeletionTimestamp != nil {
		// Its deletion requeues the pool.
		return 0, nil
	}
	now := c.now()
	if p.idleSince.IsZero() {
		p.idleSince = now
	}
	if left := p.idleSince.Add(c.retention(deploy)).Sub(now); left > 0 {
		return left, nil
	}
	// The cache may not have seen a TaskSet created just now; deleting the
	// pool then would cost its next task a cold start.
	namespace, hash := splitKey(key)
	busy, err := c.hasTaskSets(ctx, namespace, hash)
	if err != nil {
		return 0, err
	}
	if busy {
		p.idleSince = now
		return c.retention(deploy), nil
	}
	background := metav1.DeletePropagationBackground
	err = c.KubeClient.AppsV1().Deployments(namespace).Delete(ctx, deploy.Name, metav1.DeleteOptions{
		Preconditions:     &metav1.Preconditions{UID: &deploy.UID},
		PropagationPolicy: &background,
	})
	switch {
	case apierr.IsNotFound(err), apierr.IsConflict(err):
		// Gone already, or replaced by one the cache will bring.
		return 0, nil
	case err != nil:
		return 0, err
	}
	logging.RequireLoggerFromContext(ctx).WithField("deployment", deploy.Name).Info(ctx, "Deleted idle pool")
	return 0, nil
}

// hasTaskSets lists the pool's TaskSets live, so as not to rely on the cache.
func (c *Controller) hasTaskSets(ctx context.Context, namespace, hash string) (bool, error) {
	hashReq, err := labels.NewRequirement(common.LabelKeyPoolHash, selection.Equals, []string{hash})
	if err != nil {
		return false, err
	}
	selector := labels.NewSelector().Add(*hashReq, util.InstanceIDRequirement(c.InstanceID))
	list, err := c.WfClient.ArgoprojV1alpha1().WorkflowTaskSets(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String(), Limit: 1})
	if err != nil {
		return false, err
	}
	return len(list.Items) > 0, nil
}

// retention is how long the pool may be idle before its Deployment is
// deleted: the annotation on the Deployment, or the controller's default.
func (c *Controller) retention(deploy *appsv1.Deployment) time.Duration {
	if s, err := strconv.ParseInt(deploy.Annotations[common.AnnotationKeyPoolRetentionSeconds], 10, 64); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	return c.DefaultRetention()
}

// reconcileBusy handles a pool with TaskSets: it creates the pool's
// Deployment if there is none, and otherwise keeps its retention annotation
// that of the newest TaskSet (retentionSeconds is last writer wins). The
// Deployment of a pool another controller version wrote is left as it is.
func (c *Controller) reconcileBusy(ctx context.Context, key string, taskSets []any, deploy *appsv1.Deployment) (time.Duration, error) {
	p := c.state(key)
	p.idleSince = time.Time{}
	if deploy != nil && deploy.DeletionTimestamp != nil {
		// A pool used again while its Deployment is being deleted gets a new
		// one once the old one is gone; its deletion requeues the pool.
		return 0, nil
	}
	newest := newestTaskSet(taskSets)
	if newest == nil {
		return 0, nil
	}
	if newest.GetUID() != p.specFrom {
		read, err := c.readSpec(ctx, key, p, newest)
		if err != nil || !read {
			return 0, err
		}
	}
	if deploy == nil {
		return 0, c.create(ctx, key, p)
	}
	c.setHealth(ctx, key, podHealth(deploy, c.cachedPods(key)), deploymentRef(deploy))
	if !c.ownVersion(p.spec) {
		return 0, nil
	}
	return 0, c.recordRetention(ctx, deploy, p.spec)
}

// ownVersion reports whether this controller's version wrote spec.
func (c *Controller) ownVersion(spec *wfv1.PoolSpec) bool {
	return spec != nil && spec.ControllerVersion == c.ControllerVersion
}

// newestTaskSet is the most recently created of the TaskSets.
func newestTaskSet(taskSets []any) metav1.Object {
	var newest metav1.Object
	for _, obj := range taskSets {
		m, err := meta.Accessor(obj)
		if err != nil {
			continue
		}
		if newest == nil {
			newest = m
			continue
		}
		if c := m.GetCreationTimestamp().Compare(newest.GetCreationTimestamp().Time); c > 0 || (c == 0 && m.GetName() > newest.GetName()) {
			newest = m
		}
	}
	return newest
}

// readSpec reads the spec.pool of the TaskSet live. It reports false, with no
// error, if the TaskSet has gone, which requeues the pool.
func (c *Controller) readSpec(ctx context.Context, key string, p *pool, taskSet metav1.Object) (bool, error) {
	_, hash := splitKey(key)
	live, err := c.WfClient.ArgoprojV1alpha1().WorkflowTaskSets(taskSet.GetNamespace()).Get(ctx, taskSet.GetName(), metav1.GetOptions{})
	switch {
	case apierr.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	case live.Labels[common.LabelKeyPoolHash] != hash:
		// Replaced by a TaskSet of another pool since it was cached.
		return false, nil
	}
	p.specFrom = live.UID
	p.spec = live.Spec.Pool
	p.taskSet = &apiv1.ObjectReference{
		Kind:       workflow.WorkflowTaskSetKind,
		APIVersion: workflow.APIVersion,
		Namespace:  live.Namespace,
		Name:       live.Name,
		UID:        live.UID,
	}
	return true, nil
}

// create creates the pool's Deployment from the spec.pool last read. The pool
// is unbuildable if the spec is missing, another instance's or can't be built,
// if the API server rejects the Deployment as invalid, or if a Deployment of
// the same name isn't the pool's. A spec another controller version wrote is
// never built, and the pool is degraded: the operator enqueues its unclaimed
// tasks again for this version, and a task a worker released waits for its
// own version's pool. A transient error is returned with the pool's health
// unchanged; any other create failure, such as Forbidden (RBAC, admission or
// quota), makes the pool degraded. None ever leaves another Deployment
// serving the pool.
func (c *Controller) create(ctx context.Context, key string, p *pool) error {
	namespace, hash := splitKey(key)
	spec := p.spec
	switch {
	case spec == nil:
		c.setHealth(ctx, key, unbuildable("the WorkflowTaskSet %s has no spec.pool; install the current CRDs", p.taskSet.Name), p.taskSet)
		return nil
	case spec.InstanceID != c.InstanceID:
		c.setHealth(ctx, key, unbuildable("the pool spec is for controller instance %q, not %q", spec.InstanceID, c.InstanceID), p.taskSet)
		return nil
	case !c.ownVersion(spec):
		c.setHealth(ctx, key, dispatch.Health{State: dispatch.HealthDegraded, Reason: fmt.Sprintf(
			"the pool spec is for controller version %q, and this controller, %q, builds pools only for its own version", spec.ControllerVersion, c.ControllerVersion)}, p.taskSet)
		return nil
	}
	built, err := c.Build(namespace, hash, spec)
	if err != nil {
		c.setHealth(ctx, key, unbuildable("can't build pool %q: %v", spec.Name, err), p.taskSet)
		return nil
	}
	if built.Annotations == nil {
		built.Annotations = map[string]string{}
	}
	built.Annotations[common.AnnotationKeyPoolRetentionSeconds] = strconv.FormatInt(spec.RetentionSeconds, 10)
	deployments := c.KubeClient.AppsV1().Deployments(namespace)
	_, err = deployments.Create(ctx, built, metav1.CreateOptions{})
	switch {
	case err == nil:
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"deployment": built.Name, "replicas": spec.InitialReplicas}).Info(ctx, "Created pool")
		c.setHealth(ctx, key, dispatch.Health{State: dispatch.HealthOK}, p.taskSet)
		return nil
	case apierr.IsAlreadyExists(err):
		return c.adopt(ctx, key, p, built.Name)
	case apierr.IsInvalid(err):
		c.setHealth(ctx, key, unbuildable("the Deployment of pool %q is invalid: %v", spec.Name, err), p.taskSet)
		return nil
	case errorsutil.IsTransientErr(ctx, err):
		return err
	}
	// Including Forbidden: missing RBAC, admission or quota, which an
	// administrator can fix.
	c.setHealth(ctx, key, dispatch.Health{State: dispatch.HealthDegraded, Reason: fmt.Sprintf("can't create the Deployment of pool %q: %v", spec.Name, err)}, p.taskSet)
	return err
}

// adopt checks a Deployment of the pool's Deployment name that already exists
// but isn't cached. It is the pool's if it has the pool's hash and this
// instance's ID; the cache will bring it. Otherwise the name is taken.
func (c *Controller) adopt(ctx context.Context, key string, p *pool, name string) error {
	namespace, hash := splitKey(key)
	existing, err := c.KubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case apierr.IsNotFound(err):
		return fmt.Errorf("the Deployment %s was deleted while it was created", name)
	case err != nil:
		return err
	case c.owns(existing, hash):
		return nil
	}
	c.setHealth(ctx, key, unbuildable("the Deployment %s exists and isn't this pool's: pool names %q differ only after their first 10 hash characters, or another controller made it", name, p.spec.Name), p.taskSet)
	// Try again later, in case the other Deployment goes.
	return fmt.Errorf("the Deployment %s isn't pool %s's", name, hash)
}

// owns reports whether a Deployment is that of this instance's pool with hash.
func (c *Controller) owns(deploy *appsv1.Deployment, hash string) bool {
	instanceIDReq := util.InstanceIDRequirement(c.InstanceID)
	return deploy.Labels[common.LabelKeyPoolHash] == hash && instanceIDReq.Matches(labels.Set(deploy.Labels))
}

// recordRetention sets the Deployment's retention annotation to the spec's.
func (c *Controller) recordRetention(ctx context.Context, deploy *appsv1.Deployment, spec *wfv1.PoolSpec) error {
	if spec == nil {
		return nil
	}
	want := strconv.FormatInt(spec.RetentionSeconds, 10)
	if deploy.Annotations[common.AnnotationKeyPoolRetentionSeconds] == want {
		return nil
	}
	// The UID makes the patch fail on a Deployment that replaced this one.
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{
		"uid":         deploy.UID,
		"annotations": map[string]string{common.AnnotationKeyPoolRetentionSeconds: want},
	}})
	if err != nil {
		return err
	}
	_, err = c.KubeClient.AppsV1().Deployments(deploy.Namespace).Patch(ctx, deploy.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if apierr.IsNotFound(err) {
		return nil
	}
	return err
}

func unbuildable(format string, args ...any) dispatch.Health {
	return dispatch.Health{State: dispatch.HealthUnbuildable, Reason: fmt.Sprintf(format, args...)}
}

// cachedPods are the pool's cached pods.
func (c *Controller) cachedPods(key string) []*apiv1.Pod {
	objs, _ := c.Pods.GetIndexer().ByIndex(indexes.PoolIndex, key)
	pods := make([]*apiv1.Pod, 0, len(objs))
	for _, obj := range objs {
		if pod, ok := obj.(*apiv1.Pod); ok {
			pods = append(pods, pod)
		}
	}
	return pods
}

func deploymentRef(deploy *appsv1.Deployment) *apiv1.ObjectReference {
	return &apiv1.ObjectReference{
		Kind:       "Deployment",
		APIVersion: appsv1.SchemeGroupVersion.String(),
		Namespace:  deploy.Namespace,
		Name:       deploy.Name,
		UID:        deploy.UID,
	}
}

// setHealth records the pool's health. On a change it emits a warning event
// on involved if the pool isn't OK, and calls OnHealthChange.
func (c *Controller) setHealth(ctx context.Context, key string, health dispatch.Health, involved *apiv1.ObjectReference) {
	c.mu.Lock()
	p, ok := c.pools[key]
	if !ok {
		c.mu.Unlock()
		return
	}
	old := p.health
	p.health = health
	c.mu.Unlock()
	if old == health {
		return
	}
	namespace, hash := splitKey(key)
	logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"health": health.State, "reason": health.Reason, "was": old.State}).Info(ctx, "Pool health changed")
	if health.State != dispatch.HealthOK && involved != nil {
		c.Events.Get(ctx, namespace).Event(involved, apiv1.EventTypeWarning, "Pool"+string(health.State), health.Reason)
	}
	if c.OnHealthChange != nil {
		c.OnHealthChange(namespace, hash)
	}
}
