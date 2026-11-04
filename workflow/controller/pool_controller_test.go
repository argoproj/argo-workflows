package controller

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4"
	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/poolcontroller"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

func withHotWorkers(wfc *WorkflowController) {
	wfc.Config.HotWorkers = &config.HotWorkersConfig{Enabled: true}
}

// poolActions are the calls a fake made on Deployments or to check RBAC.
func poolActions(kube *fake.Clientset) []k8stesting.Action {
	var out []k8stesting.Action
	for _, action := range kube.Actions() {
		switch action.GetResource().Resource {
		case "deployments", "selfsubjectaccessreviews":
			out = append(out, action)
		}
	}
	return out
}

func TestPoolControllerOffWithoutTheFlag(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx)
	defer cancel()

	wfc.setUpPoolController(ctx)
	wfc.runPoolController(ctx)
	require.NoError(t, wfc.registerPoolGauges())
	assert.Nil(t, wfc.poolController)
	assert.Nil(t, wfc.poolDeploymentInformer)
	assert.Nil(t, wfc.poolHealth)
	assert.Empty(t, poolActions(wfc.kubeclientset.(*fake.Clientset)), "no Deployment calls, so no Deployment RBAC")
}

func TestPoolControllerWithoutDeploymentRBAC(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx, withHotWorkers)
	defer cancel()
	wfc.kubeclientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{Status: authorizationv1.SubjectAccessReviewStatus{Allowed: false}}, nil
	})

	wfc.setUpPoolController(ctx)
	wfc.runPoolController(ctx)
	assert.Nil(t, wfc.poolController)
	assert.Nil(t, wfc.poolDeploymentInformer)
	require.NotNil(t, wfc.poolHealth)
	health, ok := wfc.poolHealth.Health("ns", "any-hash")
	assert.True(t, ok)
	assert.Equal(t, dispatch.Health{State: dispatch.HealthDegraded, Reason: noDeploymentRBAC}, health)
	for _, action := range poolActions(wfc.kubeclientset.(*fake.Clientset)) {
		assert.Equal(t, "selfsubjectaccessreviews", action.GetResource().Resource)
	}
}

func TestPoolControllerCreatesPoolForTaskSet(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx, withHotWorkers)
	defer cancel()
	kube := wfc.kubeclientset.(*fake.Clientset)
	allowSelfSubjectAccessReviews(kube)

	wfc.setUpPoolController(ctx)
	require.NotNil(t, wfc.poolController)
	assert.Same(t, wfc.poolController, wfc.poolHealth)
	wfc.runPoolController(ctx)

	in, x := poolDeploymentTestInputs()
	// Written by this controller's version, which only builds its own pools.
	x.controllerVersion = argo.GetVersion().Version
	spec, hash := mustPoolDeploymentSpec(ctx, t, in, x)
	taskSet := &wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "node-1-task",
			UID:       "uid-node-1",
			Labels:    map[string]string{common.LabelKeyWorkflow: "wf", common.LabelKeyPoolName: spec.Name, common.LabelKeyPoolHash: hash},
		},
		Spec: wfv1.WorkflowTaskSetSpec{Pool: spec},
	}
	_, err := wfc.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("ns").Create(ctx, taskSet, metav1.CreateOptions{})
	require.NoError(t, err)

	want, err := buildPoolDeployment("ns", hash, spec)
	require.NoError(t, err)
	var got *appsv1.Deployment
	require.Eventually(t, func() bool {
		got, err = kube.AppsV1().Deployments("ns").Get(ctx, want.Name, metav1.GetOptions{})
		return err == nil
	}, 10*time.Second, 10*time.Millisecond)
	assert.Equal(t, want.Spec, got.Spec, "built from the TaskSet's spec.pool")
	assert.Equal(t, int32(2), *got.Spec.Replicas)
	assert.Equal(t, "600", got.Annotations[common.AnnotationKeyPoolRetentionSeconds])
	assert.Empty(t, got.OwnerReferences)
	health, ok := wfc.poolHealth.Health("ns", hash)
	assert.True(t, ok)
	assert.Equal(t, dispatch.HealthOK, health.State)
}

func TestTransformPoolDeployment(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	_, deploy := mustPoolDeployment(ctx, t)
	deploy.UID = "uid"
	deploy.ResourceVersion = "3"
	deploy.Annotations = map[string]string{common.AnnotationKeyPoolRetentionSeconds: "60"}
	deploy.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "kube-controller-manager"}}
	deploy.Status = appsv1.DeploymentStatus{ReadyReplicas: 1, Conditions: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentReplicaFailure}}}

	out, err := transformPoolDeployment(deploy)
	require.NoError(t, err)
	got := out.(*appsv1.Deployment)
	assert.Equal(t, deploy.Name, got.Name)
	assert.Equal(t, types.UID("uid"), got.UID)
	assert.Equal(t, "3", got.ResourceVersion)
	assert.Equal(t, deploy.Labels, got.Labels)
	assert.Equal(t, deploy.Annotations, got.Annotations)
	assert.Equal(t, deploy.Status, got.Status)
	assert.Empty(t, got.ManagedFields)
	assert.Equal(t, appsv1.DeploymentSpec{}, got.Spec)
}

func TestPoolSelector(t *testing.T) {
	assert.Equal(t, "!workflows.argoproj.io/controller-instanceid,workflows.argoproj.io/pool-hash", newInformerTestController("").poolSelector())
	assert.Equal(t, "workflows.argoproj.io/controller-instanceid=mine,workflows.argoproj.io/pool-hash", newInformerTestController("mine").poolSelector())
}

func TestPoolHealthChangeRequeuesWaitingWorkflows(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfc := newInformerTestController("")
	wfc.wfclientset = fakewfclientset.NewClientset()
	wfc.taskResultInformer = wfc.newWorkflowTaskResultInformer(ctx)
	wfc.pooledTaskSetInformer = newTestPooledTaskSetInformer(wfc)
	inPool := map[string]string{common.LabelKeyPoolHash: "hash-a"}
	for _, taskSet := range []*metav1.PartialObjectMetadata{
		pooledTaskSetMeta("queued-1", "waiting-wf", inPool),
		pooledTaskSetMeta("queued-2", "waiting-wf", inPool),
		pooledTaskSetMeta("claimed", "running-wf", inPool),
		pooledTaskSetMeta("other-pool", "other-pool-wf", nil),
	} {
		require.NoError(t, wfc.pooledTaskSetInformer.GetIndexer().Add(taskSet))
	}
	other := pooledTaskSetMeta("queued", "other-ns-wf", inPool)
	other.Namespace = "other-ns"
	require.NoError(t, wfc.pooledTaskSetInformer.GetIndexer().Add(other))
	require.NoError(t, wfc.taskResultInformer.GetIndexer().Add(poolResult("claimed", "uid-claimed", nil)))

	wfc.enqueueWorkflowsWaitingOnPool("ns", "hash-a")
	assert.Equal(t, []string{"ns/waiting-wf"}, queued(t, wfc.wfQueue), "once per workflow, and only for unclaimed tasks")

	wfc.enqueueWorkflowsWaitingOnPool("ns", "hash-unknown")
	assertNothingQueued(t, wfc.wfQueue)
}

func TestWarnPoolControllerFlagChange(t *testing.T) {
	hook := logging.NewTestHook()
	ctx := logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Info, logging.Text, hook))
	wfc := newInformerTestController("")
	// Not decided yet: nothing to compare with.
	withHotWorkers(wfc)
	wfc.warnPoolControllerFlagChange(ctx)
	assert.Empty(t, hook.AllEntries())
	wfc.Config.HotWorkers = nil
	wfc.setUpPoolController(ctx)
	assert.Equal(t, poolControllerOff, wfc.poolControllerState.Load())
	hook.Reset()
	// Unchanged: nothing to warn about.
	wfc.warnPoolControllerFlagChange(ctx)
	assert.Empty(t, hook.AllEntries())
	withHotWorkers(wfc)
	// Only a log line; the state the pool controller started with stays.
	wfc.warnPoolControllerFlagChange(ctx)
	assert.Equal(t, []logging.TestEntry{{
		Level:  logging.Warn,
		Msg:    "hotWorkers.enabled changed: dispatch follows it now, but the pool controller starts or stops only when the workflow controller restarts",
		Fields: logging.Fields{"enabled": true},
	}}, hook.AllEntries())
	assert.Equal(t, poolControllerOff, wfc.poolControllerState.Load())
	assert.Nil(t, wfc.poolController)
}

// TestHotWorkersRBACComponent checks the optional RBAC component grants the
// controller what the pool controller uses, and nothing more.
func TestHotWorkersRBACComponent(t *testing.T) {
	data, err := os.ReadFile("../../manifests/components/hot-workers/workflow-controller-rbac.yaml")
	require.NoError(t, err)
	var patch []struct {
		Op    string            `json:"op"`
		Path  string            `json:"path"`
		Value rbacv1.PolicyRule `json:"value"`
	}
	require.NoError(t, yaml.UnmarshalStrict(data, &patch))
	require.Len(t, patch, 1)
	assert.Equal(t, "add", patch[0].Op)
	assert.Equal(t, "/rules/-", patch[0].Path)
	assert.Equal(t, rbacv1.PolicyRule{
		APIGroups: []string{appsv1.GroupName},
		Resources: []string{"deployments"},
		Verbs:     poolcontroller.DeploymentVerbs,
	}, patch[0].Value)
}
