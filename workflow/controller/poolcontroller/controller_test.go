package poolcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

const (
	testNamespace = "ns"
	testHash      = "0123456789abcdef"
	// otherHash shares testHash's first 10 characters, so its pool of the
	// same name has the same Deployment name.
	otherHash = "0123456789ffffff"
	// testVersion is the harness controller's version.
	testVersion = "v4.1.0"
)

var errTestBuild = errors.New("can't build this")

type testEventRecorderManager struct{ recorder *record.FakeRecorder }

func (m testEventRecorderManager) Get(context.Context, string) record.EventRecorder {
	return m.recorder
}

type harness struct {
	t        *testing.T
	c        *Controller
	kube     *fake.Clientset
	wf       *fakewfclientset.Clientset
	events   *record.FakeRecorder
	now      time.Time
	changes  []string
	builtFor []*wfv1.PoolSpec
}

func (h *harness) ctx() context.Context {
	return logging.TestContext(h.t.Context())
}

func newInformer(obj runtime.Object) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(&cache.ListWatch{}, obj, 0, cache.Indexers{
		indexes.PoolIndex: indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyPoolHash),
	})
}

// newHarness is a controller of instanceID whose informers are never run:
// tests fill their caches and reconcile by hand.
func newHarness(t *testing.T, instanceID string) *harness {
	h := &harness{
		t:      t,
		kube:   fake.NewClientset(),
		wf:     fakewfclientset.NewClientset(),
		events: record.NewFakeRecorder(16),
		now:    time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC),
	}
	queue := workqueue.NewTypedRateLimitingQueue(NewRateLimiter())
	t.Cleanup(queue.ShutDown)
	h.c = New(Options{
		KubeClient:        h.kube,
		WfClient:          h.wf,
		TaskSets:          newInformer(&metav1.PartialObjectMetadata{}),
		Pods:              newInformer(&apiv1.Pod{}),
		Deployments:       newInformer(&appsv1.Deployment{}),
		Queue:             queue,
		InstanceID:        instanceID,
		ControllerVersion: testVersion,
		Build:             h.build,
		DefaultRetention:  func() time.Duration { return 10 * time.Minute },
		Events:            testEventRecorderManager{recorder: h.events},
		OnHealthChange: func(namespace, hash string) {
			h.changes = append(h.changes, namespace+"/"+hash)
		},
	})
	h.c.now = func() time.Time { return h.now }
	return h
}

// build stands in for buildPoolDeployment: it fails for a pool named "bad".
func (h *harness) build(namespace, hash string, spec *wfv1.PoolSpec) (*appsv1.Deployment, error) {
	h.builtFor = append(h.builtFor, spec)
	if spec.Name == "bad" {
		return nil, errTestBuild
	}
	labels := map[string]string{common.LabelKeyPoolName: spec.Name, common.LabelKeyPoolHash: hash}
	if spec.InstanceID != "" {
		labels[common.LabelKeyControllerInstanceID] = spec.InstanceID
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name + "-" + hash[:10], Namespace: namespace, Labels: labels},
		Spec:       appsv1.DeploymentSpec{Replicas: new(spec.InitialReplicas)},
	}, nil
}

func poolSpec(name string, replicas int32, retention int64) *wfv1.PoolSpec {
	return &wfv1.PoolSpec{Name: name, InitialReplicas: replicas, RetentionSeconds: retention, ControllerVersion: testVersion}
}

// addTaskSet stores a pooled TaskSet of the pool with testHash in namespace, in
// the API and, unless uncached, in the cache, created at the harness's now.
func (h *harness) addTaskSet(namespace, nodeID string, spec *wfv1.PoolSpec, uncached ...bool) *wfv1.WorkflowTaskSet {
	h.t.Helper()
	taskSet := &wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         namespace,
			Name:              nodeID + "-task",
			UID:               types.UID("uid-" + namespace + "-" + nodeID),
			CreationTimestamp: metav1.NewTime(h.now),
			Labels:            map[string]string{common.LabelKeyWorkflow: "wf", common.LabelKeyPoolHash: testHash},
		},
		Spec: wfv1.WorkflowTaskSetSpec{Pool: spec},
	}
	if spec != nil && spec.InstanceID != "" {
		taskSet.Labels[common.LabelKeyControllerInstanceID] = spec.InstanceID
	}
	_, err := h.wf.ArgoprojV1alpha1().WorkflowTaskSets(namespace).Create(h.ctx(), taskSet, metav1.CreateOptions{})
	require.NoError(h.t, err)
	if len(uncached) == 0 || !uncached[0] {
		require.NoError(h.t, h.c.TaskSets.GetIndexer().Add(&metav1.PartialObjectMetadata{ObjectMeta: taskSet.ObjectMeta}))
	}
	return taskSet
}

func (h *harness) deleteTaskSet(taskSet *wfv1.WorkflowTaskSet) {
	h.t.Helper()
	require.NoError(h.t, h.wf.ArgoprojV1alpha1().WorkflowTaskSets(taskSet.Namespace).Delete(h.ctx(), taskSet.Name, metav1.DeleteOptions{}))
	require.NoError(h.t, h.c.TaskSets.GetIndexer().Delete(&metav1.PartialObjectMetadata{ObjectMeta: taskSet.ObjectMeta}))
}

// cacheCreated puts the Deployments created so far into the cache, as the
// informer would, giving each a UID.
func (h *harness) cacheCreated() {
	h.t.Helper()
	for _, action := range h.kube.Actions() {
		if create, ok := action.(k8stesting.CreateAction); ok && action.GetResource().Resource == "deployments" {
			deploy := create.GetObject().(*appsv1.Deployment).DeepCopy()
			deploy.UID = types.UID("uid-" + deploy.Name)
			require.NoError(h.t, h.c.Deployments.GetIndexer().Add(deploy))
		}
	}
}

// reconcile reconciles the pool with testHash in namespace.
func (h *harness) reconcile(namespace string) (time.Duration, error) {
	return h.c.reconcile(h.ctx(), indexes.MetaNamespaceLabelIndex(namespace, testHash))
}

func (h *harness) mustReconcile(namespace string) time.Duration {
	h.t.Helper()
	after, err := h.reconcile(namespace)
	require.NoError(h.t, err)
	return after
}

// deploymentActions are the API calls made on Deployments, as verb and name.
func (h *harness) deploymentActions() []string {
	var out []string
	for _, action := range h.kube.Actions() {
		if action.GetResource().Resource != "deployments" {
			continue
		}
		name := ""
		switch a := action.(type) {
		case k8stesting.CreateAction:
			name = a.GetObject().(*appsv1.Deployment).Name
		case k8stesting.PatchAction:
			name = a.GetName()
		case k8stesting.DeleteAction:
			name = a.GetName()
		case k8stesting.GetAction:
			name = a.GetName()
		}
		out = append(out, action.GetVerb()+" "+name)
	}
	return out
}

func (h *harness) createdDeployments() []*appsv1.Deployment {
	var out []*appsv1.Deployment
	for _, action := range h.kube.Actions() {
		if create, ok := action.(k8stesting.CreateAction); ok && action.GetResource().Resource == "deployments" {
			out = append(out, create.GetObject().(*appsv1.Deployment))
		}
	}
	return out
}

// health is the health of the pool with testHash in namespace.
func (h *harness) health(namespace string) dispatch.Health {
	h.t.Helper()
	health, ok := h.c.Health(namespace, testHash)
	require.True(h.t, ok)
	return health
}

func (h *harness) drainEvents() []string {
	var out []string
	for {
		select {
		case e := <-h.events.Events:
			out = append(out, e)
		default:
			return out
		}
	}
}

func TestCreatesDeploymentAtInitialReplicas(t *testing.T) {
	h := newHarness(t, "")
	first := poolSpec("my-pool", 3, 120)
	h.addTaskSet(testNamespace, "node-1", first)

	h.mustReconcile(testNamespace)
	assert.Equal(t, []*wfv1.PoolSpec{first}, h.builtFor, "built from the TaskSet's spec.pool exactly")
	created := h.createdDeployments()
	require.Len(t, created, 1)
	want, err := h.build(testNamespace, testHash, first)
	require.NoError(t, err)
	want.Annotations = map[string]string{common.AnnotationKeyPoolRetentionSeconds: "120"}
	assert.Equal(t, want, created[0], "what was built, with the spec's retention")
	assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))

	// A later TaskSet asking for other replicas changes nothing once the
	// Deployment exists; its retention, the last writer's, is recorded.
	h.cacheCreated()
	h.kube.ClearActions()
	h.now = h.now.Add(time.Second)
	h.addTaskSet(testNamespace, "node-2", poolSpec("my-pool", 7, 30))
	h.mustReconcile(testNamespace)
	assert.Equal(t, []string{"patch my-pool-0123456789"}, h.deploymentActions())
	patch := h.kube.Actions()[0].(k8stesting.PatchAction)
	assert.Equal(t, types.MergePatchType, patch.GetPatchType())
	var body map[string]map[string]any
	require.NoError(t, json.Unmarshal(patch.GetPatch(), &body))
	assert.Equal(t, map[string]any{
		"uid":         "uid-my-pool-0123456789",
		"annotations": map[string]any{common.AnnotationKeyPoolRetentionSeconds: "30"},
	}, body["metadata"], "only the annotation, on that Deployment only")
	deploy, err := h.kube.AppsV1().Deployments(testNamespace).Get(h.ctx(), "my-pool-0123456789", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, int32(3), *deploy.Spec.Replicas)

	// Nothing new: no API calls.
	h.kube.ClearActions()
	h.wf.ClearActions()
	require.NoError(t, h.c.Deployments.GetIndexer().Update(deploy))
	h.mustReconcile(testNamespace)
	assert.Empty(t, h.deploymentActions())
	assert.Empty(t, h.wf.Actions())
}

func TestDeletesIdlePoolAfterRetention(t *testing.T) {
	h := newHarness(t, "")
	taskSet := h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
	h.mustReconcile(testNamespace)
	h.cacheCreated()
	h.deleteTaskSet(taskSet)
	h.kube.ClearActions()

	assert.Equal(t, 60*time.Second, h.mustReconcile(testNamespace))
	h.now = h.now.Add(59 * time.Second)
	assert.Equal(t, time.Second, h.mustReconcile(testNamespace))
	assert.Empty(t, h.deploymentActions())

	h.now = h.now.Add(time.Second)
	assert.Zero(t, h.mustReconcile(testNamespace))
	require.Equal(t, []string{"delete my-pool-0123456789"}, h.deploymentActions())
	opts := h.kube.Actions()[len(h.kube.Actions())-1].(k8stesting.DeleteAction).GetDeleteOptions()
	assert.Equal(t, types.UID("uid-my-pool-0123456789"), *opts.Preconditions.UID)

	// Once its deletion reaches the cache, the pool is forgotten.
	deploy, _, _ := h.c.Deployments.GetIndexer().GetByKey(testNamespace + "/my-pool-0123456789")
	require.NoError(t, h.c.Deployments.GetIndexer().Delete(deploy))
	h.mustReconcile(testNamespace)
	_, known := h.c.Health(testNamespace, testHash)
	assert.False(t, known)
}

func TestIdleRetentionRestartsWithTheController(t *testing.T) {
	h := newHarness(t, "")
	h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
	h.mustReconcile(testNamespace)
	created := h.createdDeployments()[0].DeepCopy()
	created.UID = "uid"

	// A new controller long after the pool went idle: its timer starts over.
	restarted := newHarness(t, "")
	restarted.now = h.now.Add(time.Hour)
	require.NoError(t, restarted.c.Deployments.GetIndexer().Add(created))
	assert.Equal(t, 60*time.Second, restarted.mustReconcile(testNamespace))
	assert.Empty(t, restarted.deploymentActions())
}

func TestIdlePoolUsesDefaultRetention(t *testing.T) {
	h := newHarness(t, "")
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace, Name: "old-pool-0123456789", UID: "uid",
		Labels: map[string]string{common.LabelKeyPoolHash: testHash},
	}}
	require.NoError(t, h.c.Deployments.GetIndexer().Add(deploy))
	assert.Equal(t, 10*time.Minute, h.mustReconcile(testNamespace))
}

func TestIdlePoolNotDeletedWhileTaskSetUncached(t *testing.T) {
	h := newHarness(t, "")
	taskSet := h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
	h.mustReconcile(testNamespace)
	h.cacheCreated()
	h.deleteTaskSet(taskSet)
	// Created, but not yet in the cache.
	h.addTaskSet(testNamespace, "node-2", poolSpec("my-pool", 1, 60), true)
	h.kube.ClearActions()

	assert.Equal(t, time.Minute, h.mustReconcile(testNamespace))
	h.now = h.now.Add(time.Minute)
	assert.Equal(t, time.Minute, h.mustReconcile(testNamespace), "its retention starts over")
	h.now = h.now.Add(59 * time.Second)
	assert.Equal(t, time.Second, h.mustReconcile(testNamespace))
	assert.Empty(t, h.deploymentActions())
}

func TestRecreatesDeploymentDeletedWhileInUse(t *testing.T) {
	h := newHarness(t, "")
	h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
	h.mustReconcile(testNamespace)
	h.cacheCreated()
	deploy, _, _ := h.c.Deployments.GetIndexer().GetByKey(testNamespace + "/my-pool-0123456789")
	deleting := deploy.(*appsv1.Deployment).DeepCopy()
	deleting.DeletionTimestamp = new(metav1.NewTime(h.now))
	require.NoError(t, h.c.Deployments.GetIndexer().Update(deleting))
	require.NoError(t, h.kube.AppsV1().Deployments(testNamespace).Delete(h.ctx(), deleting.Name, metav1.DeleteOptions{}))
	h.kube.ClearActions()
	h.wf.ClearActions()

	// Its name is still taken while it is being deleted, and it isn't
	// changed, even for a newer TaskSet's retention.
	h.now = h.now.Add(time.Second)
	h.addTaskSet(testNamespace, "node-2", poolSpec("my-pool", 1, 30))
	h.wf.ClearActions()
	h.mustReconcile(testNamespace)
	assert.Empty(t, h.deploymentActions())
	assert.Empty(t, h.wf.Actions())

	require.NoError(t, h.c.Deployments.GetIndexer().Delete(deleting))
	h.mustReconcile(testNamespace)
	assert.Equal(t, []string{"create my-pool-0123456789"}, h.deploymentActions())
}

func TestUnbuildablePools(t *testing.T) {
	foreign := func(labels map[string]string) *appsv1.Deployment {
		return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "my-pool-0123456789", Labels: labels}}
	}
	for _, tt := range []struct {
		name       string
		instanceID string
		spec       *wfv1.PoolSpec
		existing   *appsv1.Deployment
		createErr  error
		wantErr    bool
		reason     string
	}{
		{name: "no spec.pool", reason: "the WorkflowTaskSet node-1-task has no spec.pool; install the current CRDs"},
		{name: "builder rejects the spec", spec: poolSpec("bad", 1, 60), reason: `can't build pool "bad": can't build this`},
		{name: "spec for another instance", instanceID: "mine", spec: &wfv1.PoolSpec{Name: "my-pool", InstanceID: "theirs"}, reason: `the pool spec is for controller instance "theirs", not "mine"`},
		{
			name:      "API server rejects the Deployment",
			spec:      poolSpec("my-pool", 1, 60),
			createErr: apierr.NewInvalid(schema.GroupKind{Group: "apps", Kind: "Deployment"}, "my-pool-0123456789", nil),
			reason:    `the Deployment of pool "my-pool" is invalid: Deployment.apps "my-pool-0123456789" is invalid`,
		},
		{
			name:     "name taken by a pool with another hash",
			spec:     poolSpec("my-pool", 1, 60),
			existing: foreign(map[string]string{common.LabelKeyPoolHash: otherHash}),
			wantErr:  true,
			reason:   `the Deployment my-pool-0123456789 exists and isn't this pool's: pool names "my-pool" differ only after their first 10 hash characters, or another controller made it`,
		},
		{
			name:     "name taken by another instance's pool",
			spec:     poolSpec("my-pool", 1, 60),
			existing: foreign(map[string]string{common.LabelKeyPoolHash: testHash, common.LabelKeyControllerInstanceID: "other"}),
			wantErr:  true,
			reason:   `the Deployment my-pool-0123456789 exists and isn't this pool's: pool names "my-pool" differ only after their first 10 hash characters, or another controller made it`,
		},
		{
			name:     "name taken by a Deployment that isn't a pool's",
			spec:     poolSpec("my-pool", 1, 60),
			existing: foreign(nil),
			wantErr:  true,
			reason:   `the Deployment my-pool-0123456789 exists and isn't this pool's: pool names "my-pool" differ only after their first 10 hash characters, or another controller made it`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, tt.instanceID)
			if tt.existing != nil {
				_, err := h.kube.AppsV1().Deployments(testNamespace).Create(h.ctx(), tt.existing, metav1.CreateOptions{})
				require.NoError(t, err)
				h.kube.ClearActions()
			}
			if tt.createErr != nil {
				h.kube.PrependReactor("create", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, tt.createErr
				})
			}
			spec := tt.spec
			if spec != nil && tt.instanceID != "" && spec.InstanceID == "" {
				spec.InstanceID = tt.instanceID
			}
			h.addTaskSet(testNamespace, "node-1", spec)

			_, err := h.reconcile(testNamespace)
			if tt.wantErr {
				require.Error(t, err, "retried, in case the name is freed")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, dispatch.Health{State: dispatch.HealthUnbuildable, Reason: tt.reason}, h.health(testNamespace))
			assert.Equal(t, []string{"Warning PoolUnbuildable " + tt.reason}, h.drainEvents())
			assert.Equal(t, []string{testNamespace + "/" + testHash}, h.changes)
			for _, action := range h.deploymentActions() {
				assert.Contains(t, []string{"create my-pool-0123456789", "get my-pool-0123456789"}, action, "never changes another Deployment")
			}

			// Reconciling again changes nothing, so nothing is reported again.
			_, _ = h.reconcile(testNamespace)
			assert.Empty(t, h.drainEvents())
			assert.Len(t, h.changes, 1)
		})
	}
}

// TestOtherVersionPools checks that a pool whose spec.pool another
// controller version wrote is never built or changed: without a Deployment it
// is degraded, saying why, and its existing Deployment is left as it is, for
// its retention to delete, while its health is still reported.
func TestOtherVersionPools(t *testing.T) {
	older := func(retention int64) *wfv1.PoolSpec {
		spec := poolSpec("my-pool", 1, retention)
		spec.ControllerVersion = "v4.0.0"
		return spec
	}
	t.Run("not built", func(t *testing.T) {
		h := newHarness(t, "")
		h.addTaskSet(testNamespace, "node-1", older(60))
		h.mustReconcile(testNamespace)
		assert.Empty(t, h.deploymentActions())
		assert.Empty(t, h.builtFor)
		reason := `the pool spec is for controller version "v4.0.0", and this controller, "v4.1.0", builds pools only for its own version`
		assert.Equal(t, dispatch.Health{State: dispatch.HealthDegraded, Reason: reason}, h.health(testNamespace))
		assert.Equal(t, []string{"Warning PoolDegraded " + reason}, h.drainEvents())
	})
	t.Run("not changed", func(t *testing.T) {
		h := newHarness(t, "")
		deploy, err := h.build(testNamespace, testHash, older(60))
		require.NoError(t, err)
		deploy.UID = "uid-old"
		deploy.Annotations = map[string]string{common.AnnotationKeyPoolRetentionSeconds: "60"}
		require.NoError(t, h.c.Deployments.GetIndexer().Add(deploy))
		h.builtFor = nil
		h.addTaskSet(testNamespace, "node-1", older(30))
		h.mustReconcile(testNamespace)
		assert.Empty(t, h.deploymentActions(), "its retention annotation isn't changed either")
		assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace), "its pods' health")

		// With no TaskSets left its retention, as it was built, runs out.
		h.deleteTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "node-1-task", UID: "uid-ns-node-1"}})
		assert.Equal(t, 60*time.Second, h.mustReconcile(testNamespace))
		h.now = h.now.Add(time.Minute)
		h.mustReconcile(testNamespace)
		assert.Equal(t, []string{"delete " + deploy.Name}, h.deploymentActions())
	})
}

func TestCreateFailures(t *testing.T) {
	forbidden := apierr.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "my-pool-0123456789", errors.New("no RBAC"))
	t.Run("forbidden degrades the pool", func(t *testing.T) {
		h := newHarness(t, "")
		h.kube.PrependReactor("create", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, forbidden
		})
		h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		_, err := h.reconcile(testNamespace)
		require.Error(t, err, "retried with back-off")
		reason := `can't create the Deployment of pool "my-pool": ` + forbidden.Error()
		assert.Equal(t, dispatch.Health{State: dispatch.HealthDegraded, Reason: reason}, h.health(testNamespace))
		assert.Equal(t, []string{"Warning PoolDegraded " + reason}, h.drainEvents())

		// Once allowed, the pool recovers, without reading its TaskSet again.
		h.kube.ReactionChain = h.kube.ReactionChain[1:]
		h.wf.ClearActions()
		h.mustReconcile(testNamespace)
		assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))
		assert.Empty(t, h.wf.Actions())
		assert.Len(t, h.changes, 2)
	})
	t.Run("transient errors are retried without changing health", func(t *testing.T) {
		h := newHarness(t, "")
		h.kube.PrependReactor("create", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierr.NewServiceUnavailable("try later")
		})
		h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		_, err := h.reconcile(testNamespace)
		require.Error(t, err)
		assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))
		assert.Empty(t, h.changes)
	})
	t.Run("a Deployment of the pool the cache hasn't seen is adopted", func(t *testing.T) {
		h := newHarness(t, "")
		h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		h.mustReconcile(testNamespace)
		h.mustReconcile(testNamespace)
		assert.Equal(t, []string{"create my-pool-0123456789", "create my-pool-0123456789", "get my-pool-0123456789"}, h.deploymentActions())
		assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))
	})
	t.Run("exceeded quota degrades the pool", func(t *testing.T) {
		h := newHarness(t, "")
		quota := apierr.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "my-pool-0123456789", errors.New("exceeded quota: compute"))
		h.kube.PrependReactor("create", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, quota
		})
		h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		_, err := h.reconcile(testNamespace)
		require.Error(t, err, "retried with back-off")
		assert.Equal(t, dispatch.Health{State: dispatch.HealthDegraded, Reason: `can't create the Deployment of pool "my-pool": ` + quota.Error()}, h.health(testNamespace))
	})
	t.Run("a Deployment deleted between its create and get is retried", func(t *testing.T) {
		h := newHarness(t, "")
		h.kube.PrependReactor("create", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierr.NewAlreadyExists(schema.GroupResource{Group: "apps", Resource: "deployments"}, "my-pool-0123456789")
		})
		h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		_, err := h.reconcile(testNamespace)
		require.ErrorContains(t, err, "the Deployment my-pool-0123456789 was deleted while it was created")
		assert.Equal(t, []string{"create my-pool-0123456789", "get my-pool-0123456789"}, h.deploymentActions())
		assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))
	})
	t.Run("a TaskSet replaced by another pool's isn't read", func(t *testing.T) {
		h := newHarness(t, "")
		taskSet := h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		taskSet.Labels[common.LabelKeyPoolHash] = otherHash
		_, err := h.wf.ArgoprojV1alpha1().WorkflowTaskSets(testNamespace).Update(h.ctx(), taskSet, metav1.UpdateOptions{})
		require.NoError(t, err)
		h.mustReconcile(testNamespace)
		assert.Empty(t, h.deploymentActions())
		assert.Empty(t, h.builtFor)
	})
	t.Run("a TaskSet gone before it is read waits for its deletion", func(t *testing.T) {
		h := newHarness(t, "")
		taskSet := h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		require.NoError(t, h.wf.ArgoprojV1alpha1().WorkflowTaskSets(testNamespace).Delete(h.ctx(), taskSet.Name, metav1.DeleteOptions{}))
		h.mustReconcile(testNamespace)
		assert.Empty(t, h.deploymentActions())
	})
}

func TestInstanceIDs(t *testing.T) {
	h := newHarness(t, "mine")
	ours := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyPoolHash: testHash, common.LabelKeyControllerInstanceID: "mine"}}}
	assert.True(t, h.c.owns(ours, testHash))
	assert.False(t, h.c.owns(ours, otherHash))
	assert.False(t, newHarness(t, "").c.owns(ours, testHash))
	unlabelled := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyPoolHash: testHash}}}
	assert.True(t, newHarness(t, "").c.owns(unlabelled, testHash))
	assert.False(t, h.c.owns(unlabelled, testHash))
}

func TestSameHashInTwoNamespaces(t *testing.T) {
	h := newHarness(t, "")
	h.addTaskSet("ns1", "node-1", poolSpec("my-pool", 1, 60))
	h.addTaskSet("ns2", "node-1", poolSpec("bad", 1, 60))
	h.mustReconcile("ns1")
	h.mustReconcile("ns2")

	created := h.createdDeployments()
	require.Len(t, created, 1)
	assert.Equal(t, "ns1", created[0].Namespace)
	assert.Equal(t, dispatch.HealthOK, h.health("ns1").State)
	assert.Equal(t, dispatch.HealthUnbuildable, h.health("ns2").State)
	assert.Equal(t, []string{"ns2/" + testHash}, h.changes)
}

func readyPod(name string, ready bool) *apiv1.Pod {
	status := apiv1.ConditionFalse
	if ready {
		status = apiv1.ConditionTrue
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: name, Labels: map[string]string{common.LabelKeyPoolHash: testHash}},
		Status:     apiv1.PodStatus{Conditions: []apiv1.PodCondition{{Type: apiv1.PodReady, Status: status}}},
	}
}

func waitingPod(name, container, reason, message string) *apiv1.Pod {
	pod := readyPod(name, false)
	pod.Status.ContainerStatuses = []apiv1.ContainerStatus{{
		Name:  container,
		Image: "my-image:v1",
		State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: reason, Message: message}},
	}}
	return pod
}

func TestPodHealth(t *testing.T) {
	replicaFailure := &appsv1.Deployment{Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{{
		Type: appsv1.DeploymentReplicaFailure, Status: apiv1.ConditionTrue, Reason: "FailedCreate", Message: `pods "p-" is forbidden: exceeded quota`,
	}}}}
	deleting := waitingPod("deleting", "main", "ImagePullBackOff", "")
	deleting.DeletionTimestamp = new(metav1.Now())
	// The legacy layout's init container, which installs argoexec.
	initPullFailing := readyPod("a", false)
	initPullFailing.Status.InitContainerStatuses = []apiv1.ContainerStatus{{
		Name:  "init",
		Image: "argoexec:v4",
		State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
	}}
	initPullFailing.Status.ContainerStatuses = []apiv1.ContainerStatus{{
		Name:  "main",
		State: apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: "PodInitializing"}},
	}}
	replicaFailureResolved := &appsv1.Deployment{Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{{
		Type: appsv1.DeploymentReplicaFailure, Status: apiv1.ConditionFalse, Reason: "FailedCreate", Message: "was forbidden",
	}}}}
	ok := dispatch.Health{State: dispatch.HealthOK}
	degraded := func(reason string) dispatch.Health {
		return dispatch.Health{State: dispatch.HealthDegraded, Reason: reason}
	}
	pullFailed := degraded(`ImagePullBackOff: container "main" can't pull image "my-image:v1"`)
	for _, tt := range []struct {
		name   string
		deploy *appsv1.Deployment
		pods   []*apiv1.Pod
		want   dispatch.Health
	}{
		{name: "no pods yet", want: ok},
		{name: "starting", pods: []*apiv1.Pod{readyPod("a", false)}, want: ok},
		{name: "pull failing", pods: []*apiv1.Pod{waitingPod("a", "main", "ErrImagePull", "not found")}, want: pullFailed},
		{name: "pull backing off reads the same", pods: []*apiv1.Pod{waitingPod("a", "main", "ImagePullBackOff", `Back-off pulling image "my-image:v1"`)}, want: pullFailed},
		{name: "one pod ready serves the pool", pods: []*apiv1.Pod{waitingPod("a", "main", "ErrImagePull", ""), readyPod("b", true)}, want: ok},
		{name: "a pod being deleted doesn't count", pods: []*apiv1.Pod{deleting}, want: ok},
		{name: "missing secret", pods: []*apiv1.Pod{waitingPod("a", "main", "CreateContainerConfigError", `secret "s" not found`)}, want: degraded(`CreateContainerConfigError: container "main": secret "s" not found`)},
		{name: "crash loop", pods: []*apiv1.Pod{waitingPod("a", "worker", "CrashLoopBackOff", "back-off 10s")}, want: degraded(`CrashLoopBackOff: container "worker" keeps exiting`)},
		{name: "still creating", pods: []*apiv1.Pod{waitingPod("a", "main", "ContainerCreating", "")}, want: ok},
		{name: "pods can't be created", deploy: replicaFailure, want: degraded(`FailedCreate: pods "p-" is forbidden: exceeded quota`)},
		{name: "pods can be created again", deploy: replicaFailureResolved, want: ok},
		{name: "init container pull failing", pods: []*apiv1.Pod{initPullFailing}, want: degraded(`ImagePullBackOff: container "init" can't pull image "argoexec:v4"`)},
		{name: "invalid image", pods: []*apiv1.Pod{waitingPod("a", "main", "InvalidImageName", "bad name")}, want: degraded(`InvalidImageName: container "main": bad name`)},
		{name: "image never pulled", pods: []*apiv1.Pod{waitingPod("a", "main", "ErrImageNeverPull", "not present")}, want: degraded(`ErrImageNeverPull: container "main": not present`)},
		{name: "container can't be created", pods: []*apiv1.Pod{waitingPod("a", "main", "CreateContainerError", "no such device")}, want: degraded(`CreateContainerError: container "main": no such device`)},
		{
			name: "the same problem whichever pod reports it first",
			pods: []*apiv1.Pod{waitingPod("z", "worker", "CrashLoopBackOff", ""), waitingPod("a", "main", "ErrImagePull", "")},
			want: degraded(`CrashLoopBackOff: container "worker" keeps exiting`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			deploy := tt.deploy
			if deploy == nil {
				deploy = &appsv1.Deployment{}
			}
			assert.Equal(t, tt.want, podHealth(deploy, tt.pods))
		})
	}
}

func TestHealthChangesReportedOnce(t *testing.T) {
	h := newHarness(t, "")
	h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
	h.mustReconcile(testNamespace)
	h.cacheCreated()
	assert.Empty(t, h.changes, "a new pool is OK, as an unknown one is")

	pod := waitingPod("a", "main", "ErrImagePull", "not found")
	require.NoError(t, h.c.Pods.GetIndexer().Add(pod))
	h.mustReconcile(testNamespace)
	// The kubelet alternates the reason while it retries.
	pod = waitingPod("a", "main", "ImagePullBackOff", "Back-off")
	require.NoError(t, h.c.Pods.GetIndexer().Update(pod))
	h.mustReconcile(testNamespace)
	assert.Equal(t, dispatch.HealthDegraded, h.health(testNamespace).State)
	assert.Len(t, h.changes, 1)
	events := h.drainEvents()
	require.Len(t, events, 1)
	assert.Contains(t, events[0], "Warning PoolDegraded ")

	require.NoError(t, h.c.Pods.GetIndexer().Update(readyPod("a", true)))
	h.mustReconcile(testNamespace)
	assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))
	assert.Len(t, h.changes, 2)
	assert.Empty(t, h.drainEvents(), "recovery is no warning")
}

func TestHealthUnknownPool(t *testing.T) {
	h := newHarness(t, "")
	_, ok := h.c.Health(testNamespace, testHash)
	assert.False(t, ok)
	h.mustReconcile(testNamespace)
	_, ok = h.c.Health(testNamespace, testHash)
	assert.False(t, ok, "a pool with neither TaskSets nor a Deployment isn't remembered")
}

func TestEnqueueAfterTaskSet(t *testing.T) {
	h := newHarness(t, "")
	taskSet := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "node-1-task", Labels: map[string]string{common.LabelKeyPoolHash: testHash}}}
	h.c.enqueueAfterTaskSet(taskSet)
	assert.Equal(t, 1, h.c.Queue.Len(), "a pool with no Deployment is reconciled at once")
	key, _ := h.c.Queue.Get()
	assert.Equal(t, testNamespace+"/"+testHash, key)
	h.c.Queue.Done(key)

	require.NoError(t, h.c.Deployments.GetIndexer().Add(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace, Name: "my-pool-0123456789", Labels: map[string]string{common.LabelKeyPoolHash: testHash},
	}}))
	queue := &recordingQueue{TypedRateLimitingInterface: h.c.Queue}
	h.c.Queue = queue
	h.c.enqueueAfterTaskSet(cache.DeletedFinalStateUnknown{Obj: taskSet})
	assert.Zero(t, h.c.Queue.Len(), "a pool with a Deployment is batched")
	assert.Equal(t, []string{"after " + key + " 1s"}, queue.calls)

	h.c.enqueueAfterTaskSet(&metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "agent"}})
	assert.Zero(t, h.c.Queue.Len(), "not a pool's")
}

// recordingQueue records how processNextItem requeues or forgets a key,
// without delaying anything.
type recordingQueue struct {
	workqueue.TypedRateLimitingInterface[string]
	calls []string
}

func (q *recordingQueue) AddAfter(key string, d time.Duration) {
	q.calls = append(q.calls, "after "+key+" "+d.String())
}

func (q *recordingQueue) AddRateLimited(key string) {
	q.calls = append(q.calls, "rate limited "+key)
}

func (q *recordingQueue) Forget(key string) {
	q.calls = append(q.calls, "forget "+key)
}

// TestProcessNextItem checks that a reconcile's wait becomes a delayed requeue
// and its error a rate-limited one. The Deployment informer never resyncs, so
// the delayed requeue is all that deletes an idle pool.
func TestProcessNextItem(t *testing.T) {
	key := testNamespace + "/" + testHash
	process := func(h *harness) []string {
		h.t.Helper()
		queue := &recordingQueue{TypedRateLimitingInterface: h.c.Queue}
		h.c.Queue = queue
		queue.Add(key)
		require.True(h.t, h.c.processNextItem(h.ctx()))
		assert.Zero(h.t, queue.Len(), "nothing is requeued at once")
		return queue.calls
	}
	t.Run("an idle pool is requeued for the end of its retention, which deletes it", func(t *testing.T) {
		h := newHarness(t, "")
		taskSet := h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		h.mustReconcile(testNamespace)
		h.cacheCreated()
		h.deleteTaskSet(taskSet)
		h.kube.ClearActions()

		assert.Equal(t, []string{"forget " + key, "after " + key + " 1m0s"}, process(h))
		assert.Empty(t, h.deploymentActions())

		h.now = h.now.Add(time.Minute)
		assert.Equal(t, []string{"forget " + key}, process(h))
		assert.Equal(t, []string{"delete my-pool-0123456789"}, h.deploymentActions())
	})
	t.Run("a failure is retried with back-off", func(t *testing.T) {
		h := newHarness(t, "")
		h.kube.PrependReactor("create", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierr.NewServiceUnavailable("try later")
		})
		h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
		assert.Equal(t, []string{"rate limited " + key}, process(h))
	})
	t.Run("stops once the queue is shut down", func(t *testing.T) {
		h := newHarness(t, "")
		h.c.Queue.ShutDown()
		assert.False(t, h.c.processNextItem(h.ctx()))
	})
}

func TestRetentionAnnotation(t *testing.T) {
	h := newHarness(t, "")
	for value, want := range map[string]time.Duration{
		"0":                   0,
		"90":                  90 * time.Second,
		"-1":                  10 * time.Minute,
		"soon":                10 * time.Minute,
		"9223372036854775807": time.Duration(maxRetentionSeconds) * time.Second,
	} {
		deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{common.AnnotationKeyPoolRetentionSeconds: value}}}
		assert.Equal(t, want, h.c.retention(deploy), value)
	}
}

func TestNewestTaskSet(t *testing.T) {
	at := func(name string, created time.Time) *metav1.PartialObjectMetadata {
		return &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.NewTime(created)}}
	}
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	older, a, b := at("z", now.Add(-time.Second)), at("a", now), at("b", now)
	for _, taskSets := range [][]any{{older, a, b}, {b, a, older}, {a, older, b}} {
		assert.Equal(t, "b", newestTaskSet(taskSets).GetName(), "created at the same time: the greatest name")
	}
	assert.Nil(t, newestTaskSet(nil))
}

func TestCachedDeploymentPicksLowestName(t *testing.T) {
	h := newHarness(t, "")
	for _, name := range []string{"b-pool-0123456789", "a-pool-0123456789", "c-pool-0123456789"} {
		require.NoError(t, h.c.Deployments.GetIndexer().Add(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: name, Labels: map[string]string{common.LabelKeyPoolHash: testHash},
		}}))
	}
	deploy, err := h.c.cachedDeployment(indexes.MetaNamespaceLabelIndex(testNamespace, testHash))
	require.NoError(t, err)
	assert.Equal(t, "a-pool-0123456789", deploy.Name)
}

func TestIdlePoolIgnoresOtherInstancesTaskSets(t *testing.T) {
	h := newHarness(t, "mine")
	spec := poolSpec("my-pool", 1, 60)
	spec.InstanceID = "mine"
	taskSet := h.addTaskSet(testNamespace, "node-1", spec)
	h.mustReconcile(testNamespace)
	h.cacheCreated()
	h.deleteTaskSet(taskSet)
	theirs := poolSpec("my-pool", 1, 60)
	theirs.InstanceID = "theirs"
	h.addTaskSet(testNamespace, "node-2", theirs, true)
	h.kube.ClearActions()

	assert.Equal(t, time.Minute, h.mustReconcile(testNamespace))
	h.now = h.now.Add(time.Minute)
	h.mustReconcile(testNamespace)
	assert.Equal(t, []string{"delete my-pool-0123456789"}, h.deploymentActions())
}

func TestIdlePoolWithHugeRetentionIsKept(t *testing.T) {
	assert.Equal(t, time.Duration(maxRetentionSeconds)*time.Second, Retention(math.MaxInt64))
	assert.Positive(t, Retention(maxRetentionSeconds+1))
	h := newHarness(t, "")
	require.NoError(t, h.c.Deployments.GetIndexer().Add(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace, Name: "my-pool-0123456789", UID: "uid",
		Labels:      map[string]string{common.LabelKeyPoolHash: testHash},
		Annotations: map[string]string{common.AnnotationKeyPoolRetentionSeconds: strconv.FormatInt(math.MaxInt64, 10)},
	}}))
	assert.Positive(t, h.mustReconcile(testNamespace))
	h.now = h.now.Add(24 * time.Hour)
	assert.Positive(t, h.mustReconcile(testNamespace))
	assert.Empty(t, h.deploymentActions())
}

// idlePool is a harness with a pool whose last TaskSet has gone, and whose
// Deployment is cached.
func idlePool(t *testing.T) (*harness, *appsv1.Deployment) {
	h := newHarness(t, "")
	taskSet := h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
	h.mustReconcile(testNamespace)
	h.cacheCreated()
	h.deleteTaskSet(taskSet)
	h.kube.ClearActions()
	deploy, _, _ := h.c.Deployments.GetIndexer().GetByKey(testNamespace + "/my-pool-0123456789")
	return h, deploy.(*appsv1.Deployment)
}

func TestIdlePoolDeleteFailures(t *testing.T) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	for _, tt := range []struct {
		name    string
		err     error
		wantErr bool
	}{
		{name: "already gone", err: apierr.NewNotFound(gr, "my-pool-0123456789")},
		{name: "replaced", err: apierr.NewConflict(gr, "my-pool-0123456789", errors.New("precondition failed: UID"))},
		{name: "other errors are retried", err: apierr.NewServiceUnavailable("try later"), wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := idlePool(t)
			h.kube.PrependReactor("delete", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, tt.err
			})
			h.now = h.now.Add(time.Minute)
			h.mustReconcile(testNamespace)
			h.now = h.now.Add(time.Minute)
			after, err := h.reconcile(testNamespace)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Zero(t, after)
			assert.Equal(t, []string{"delete my-pool-0123456789"}, h.deploymentActions())
		})
	}
}

func TestIdlePoolBeingDeleted(t *testing.T) {
	h, deploy := idlePool(t)
	assert.Equal(t, time.Minute, h.mustReconcile(testNamespace))
	deleting := deploy.DeepCopy()
	deleting.DeletionTimestamp = new(metav1.NewTime(h.now))
	require.NoError(t, h.c.Deployments.GetIndexer().Update(deleting))
	require.NoError(t, h.c.Pods.GetIndexer().Add(waitingPod("a", "main", "ErrImagePull", "")))

	h.now = h.now.Add(time.Hour)
	assert.Zero(t, h.mustReconcile(testNamespace), "its deletion requeues the pool")
	assert.Empty(t, h.deploymentActions())
	assert.Equal(t, dispatch.HealthDegraded, h.health(testNamespace).State, "its health is still reported")
}

func TestIdlePoolHealth(t *testing.T) {
	h, _ := idlePool(t)
	require.NoError(t, h.c.Pods.GetIndexer().Add(waitingPod("a", "main", "ErrImagePull", "")))
	h.mustReconcile(testNamespace)
	assert.Equal(t, dispatch.Health{State: dispatch.HealthDegraded, Reason: `ImagePullBackOff: container "main" can't pull image "my-image:v1"`}, h.health(testNamespace))
}

func TestRecordRetentionFailures(t *testing.T) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	for _, tt := range []struct {
		name    string
		err     error
		wantErr bool
	}{
		{name: "gone", err: apierr.NewNotFound(gr, "my-pool-0123456789")},
		{name: "replaced, so its UID can't change", err: apierr.NewInvalid(schema.GroupKind{Group: "apps", Kind: "Deployment"}, "my-pool-0123456789", nil)},
		{name: "other errors are retried", err: apierr.NewServiceUnavailable("try later"), wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, "")
			h.addTaskSet(testNamespace, "node-1", poolSpec("my-pool", 1, 60))
			h.mustReconcile(testNamespace)
			h.cacheCreated()
			h.kube.ClearActions()
			h.kube.PrependReactor("patch", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, tt.err
			})
			h.now = h.now.Add(time.Second)
			h.addTaskSet(testNamespace, "node-2", poolSpec("my-pool", 1, 30))
			_, err := h.reconcile(testNamespace)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, []string{"patch my-pool-0123456789"}, h.deploymentActions())
			assert.Equal(t, dispatch.Health{State: dispatch.HealthOK}, h.health(testNamespace))
			assert.Empty(t, h.drainEvents())
		})
	}
}
