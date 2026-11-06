package dispatch

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestKubeCompleteIsOneTestedPatch(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	h := b.claim(t, testWorker("worker-a"), task.Key)
	b.clientset.ClearActions()

	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
	actions := b.clientset.Actions()
	require.Len(t, actions, 1)
	patch, ok := actions[0].(k8stesting.PatchAction)
	require.True(t, ok)
	var ops []jsonPatchOp
	require.NoError(t, json.Unmarshal(patch.GetPatch(), &ops))
	require.GreaterOrEqual(t, len(ops), 2)
	assert.Equal(t, jsonPatchOp{Op: "test", Path: "/metadata/labels/workflows.argoproj.io~1worker-pod-uid", Value: "worker-a-uid"}, ops[0])
	assert.Equal(t, jsonPatchOp{Op: "test", Path: "/metadata/uid", Value: string(h.Claim().UID)}, ops[1])
	assert.Contains(t, ops, jsonPatchOp{Op: "add", Path: "/metadata/labels/workflows.argoproj.io~1report-outputs-completed", Value: "true"})
}

func TestKubeCompleteNeverRecreatesTheResult(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	h := b.claim(t, testWorker("worker-a"), task.Key)
	b.deleteResult(t, task.Key)
	b.clientset.ClearActions()

	require.ErrorIs(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}), ErrClaimLost)
	for _, action := range b.clientset.Actions() {
		assert.NotEqual(t, "create", action.GetVerb())
	}
}

func TestKubeInvalidPatchOfOwnResultIsNotALostClaim(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	h := b.claim(t, testWorker("worker-a"), task.Key)
	invalid := apierr.NewInvalid(wfv1.SchemeGroupVersion.WithKind("WorkflowTaskResult").GroupKind(), "node-1", field.ErrorList{field.Invalid(field.NewPath("outputs"), nil, "bad")})
	b.setFault(func(op Op) error {
		if op.Verb == "patch" {
			return invalid
		}
		return nil
	})
	err := h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded})
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrClaimLost)
	assert.True(t, apierr.IsInvalid(err))
	select {
	case <-h.Stopped():
		t.Error("an invalid write is not a lost claim")
	default:
	}
}

// The API server reports a failed patch test as Invalid, as it does an
// invalid object, so the live result decides: one that is gone is a lost
// claim, and a read that fails keeps both errors.
func TestKubeInvalidPatchIsJudgedByTheLiveResult(t *testing.T) {
	for _, tc := range []struct {
		name    string
		readErr error
		lost    bool
	}{
		{name: "gone", readErr: apierr.NewNotFound(wfv1.SchemeGroupVersion.WithResource(ResourceResults).GroupResource(), "node-1"), lost: true},
		{name: "unreadable", readErr: errTransient},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newKubeBackend()
			ctx := logging.TestContext(t.Context())
			task := testTask("node-1")
			b.enqueue(t, task)
			h := b.claim(t, testWorker("worker-a"), task.Key)
			invalid := apierr.NewInvalid(wfv1.SchemeGroupVersion.WithKind("WorkflowTaskResult").GroupKind(), "node-1", nil)
			b.setFault(func(op Op) error {
				switch {
				case op.Verb == "patch":
					return invalid
				case op.Verb == "get" && op.Resource == ResourceResults:
					return tc.readErr
				}
				return nil
			})
			err := h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded})
			b.setFault(nil)
			assert.True(t, apierr.IsInvalid(err))
			if tc.lost {
				require.ErrorIs(t, err, ErrClaimLost)
				return
			}
			require.ErrorIs(t, err, errTransient)
			require.NotErrorIs(t, err, ErrClaimLost)
		})
	}
}

func TestKubeCandidatesNeedASource(t *testing.T) {
	b := newKubeBackend()
	_, err := b.consumer(testWorker("worker-a")).Candidates(t.Context())
	require.ErrorIs(t, err, ErrNotImplemented)

	want := []Candidate{{Key: Key{Namespace: testNamespace, NodeID: "node-1"}}}
	c := NewConsumer(b.clientset, testWorker("worker-a"), WatchSources{Candidates: staticCandidates(want)})
	got, err := c.Candidates(t.Context())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

type staticCandidates []Candidate

func (s staticCandidates) Candidates(context.Context) ([]Candidate, error) { return s, nil }

// recordingWatcher is a ClaimWatcher that hands its notify to the test.
type recordingWatcher struct {
	mu      sync.Mutex
	claims  []Claim
	notify  func(StopReason)
	cancels int
}

func (w *recordingWatcher) Watch(_ context.Context, claim Claim, notify func(StopReason)) func() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.claims = append(w.claims, claim)
	w.notify = notify
	return func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.cancels++
	}
}

func TestKubeClaimWatcherSeam(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	watcher := &recordingWatcher{}
	c := NewConsumer(b.clientset, testWorker("worker-a"), WatchSources{Claims: watcher})
	h, err := c.Claim(ctx, b.candidate(t, task.Key))
	require.NoError(t, err)
	require.Equal(t, []Claim{h.Claim()}, watcher.claims)

	watcher.notify(StopInterrupted)
	watcher.notify(StopLost)
	assertStopped(t, h, StopInterrupted)

	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed}))
	require.NoError(t, h.Release(ctx))
	assert.Equal(t, 1, watcher.cancels, "the watch is cancelled once")
}

func TestKubeClaimWatcherCancelledWhenTheClaimIsLost(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	watcher := &recordingWatcher{}
	c := NewConsumer(b.clientset, testWorker("worker-a"), WatchSources{Claims: watcher})
	h, err := c.Claim(ctx, b.candidate(t, task.Key))
	require.NoError(t, err)
	b.deleteResult(t, task.Key)
	require.ErrorIs(t, h.Progress(ctx, "1/2"), ErrClaimLost)
	assertStopped(t, h, StopLost)
	assert.Equal(t, 1, watcher.cancels)
}

// TestDispatcherShutdownRBAC checks that what the controller does to shut down
// and interrupt pooled tasks, to read a claim live, to acknowledge a finished
// node's TaskSet and to discard a result it doesn't accept, is granted by the
// shipped controller roles, the cluster role of a cluster install and the
// role of a namespace install. Both are in default installs, since
// cancellation and liveness run whether hot workers are on or not.
func TestDispatcherShutdownRBAC(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	key := func(node string) Key { return Key{Namespace: testNamespace, NodeID: node} }
	queued := b.enqueue(t, testTask("queued"))
	racing := b.enqueue(t, testTask("racing"))
	racingClaim := b.claim(t, testWorker("worker-a"), key("racing"))
	claimed := b.enqueue(t, testTask("claimed"))
	claimedClaim := b.claim(t, testWorker("worker-b"), key("claimed"))
	finished := b.enqueue(t, testTask("finished"))
	b.clientset.ClearActions()

	for _, outcome := range b.dispatcher.Shutdown(ctx, []LedgerEntry{
		{Key: key("queued"), Ledger: ledger(queued.UID, "")},
		{Key: key("racing"), Ledger: ledger(racing.UID, "")},
		{Key: key("claimed"), Ledger: ledger(claimed.UID, claimedClaim.Claim().UID)},
	}) {
		require.NoError(t, outcome.Err, outcome.NodeID)
	}
	_, err := b.dispatcher.Result(ctx, key("claimed"))
	require.NoError(t, err)
	require.NoError(t, b.dispatcher.Ack(ctx, key("finished"), finished.UID))
	require.NoError(t, b.dispatcher.Discard(ctx, key("racing"), racingClaim.Claim().UID))
	// A patch the API server rejects as invalid is told from a failed test
	// by reading the result back.
	b.setFault(func(op Op) error {
		if op.Verb == "patch" {
			return apierr.NewInvalid(wfv1.SchemeGroupVersion.WithKind("WorkflowTaskResult").GroupKind(), op.Name, nil)
		}
		return nil
	})
	require.Error(t, b.dispatcher.Interrupt(ctx, key("claimed"), claimed.UID, claimedClaim.Claim().UID))

	used := map[string][]string{}
	for _, action := range b.clientset.Actions() {
		resource := action.GetResource().Resource
		if !slices.Contains(used[resource], action.GetVerb()) {
			used[resource] = append(used[resource], action.GetVerb())
		}
	}
	require.Equal(t, []string{"delete"}, used[ResourceTaskSets])
	require.ElementsMatch(t, []string{"get", "patch", "delete"}, used[ResourceResults])
	for _, file := range []string{
		"../../manifests/cluster-install-no-crds/workflow-controller-rbac/workflow-controller-clusterrole.yaml",
		"../../manifests/namespace-install/workflow-controller-rbac/workflow-controller-role.yaml",
	} {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var role rbacv1.ClusterRole
		require.NoError(t, yaml.Unmarshal(data, &role), file)
		require.NotEmpty(t, role.Rules, file)
		for resource, verbs := range used {
			for _, verb := range verbs {
				assert.True(t, grants(role.Rules, resource, verb), "%s grants %s on %s", file, verb, resource)
			}
		}
	}
}

// grants reports whether a rule grants verb on every object of resource in
// argoproj.io.
func grants(rules []rbacv1.PolicyRule, resource, verb string) bool {
	return slices.ContainsFunc(rules, func(r rbacv1.PolicyRule) bool {
		return slices.Contains(r.APIGroups, workflow.Group) &&
			slices.Contains(r.Resources, resource) &&
			(slices.Contains(r.Verbs, verb) || slices.Contains(r.Verbs, rbacv1.VerbAll)) &&
			len(r.ResourceNames) == 0
	})
}
