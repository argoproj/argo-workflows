package dispatch

import (
	"context"
	"maps"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubefake "k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// workerCalls records the verbs a worker's calls use on each resource, as
// "verb resource".
type workerCalls struct {
	mu    sync.Mutex
	calls map[string]bool
}

func (c *workerCalls) record(verb, resource string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[verb+" "+resource] = true
}

func (c *workerCalls) sorted() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Sorted(maps.Keys(c.calls))
}

// The quick-start's hot-worker Role grants every call a worker makes, and
// nothing more, and CheckAccess checks every one of them. The calls are
// recorded from the worker's code on each backend: its Consumer and claims
// running each worker operation, and, on the Kubernetes backend, its
// Watches' metadata informers and a claim's live read of its result.
func TestWorkerRBAC(t *testing.T) {
	calls := &workerCalls{calls: map[string]bool{}}
	forEachBackend(t, func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		first, second, gone := testTask("node-1"), testTask("node-2"), testTask("node-3")
		for _, task := range []Task{first, second, gone} {
			b.enqueue(t, task)
		}
		goneCandidate := b.candidate(t, gone.Key)
		b.deleteTaskSet(t, gone.Key)
		b.setFault(func(op Op) error {
			calls.record(op.Verb, op.Resource)
			return nil
		})
		defer b.setFault(nil)
		c := b.consumer(testWorker("worker-a"))
		claimed, err := c.Claim(ctx, b.candidate(t, first.Key))
		require.NoError(t, err)
		require.NoError(t, claimed.Progress(ctx, "1/2"))
		require.NoError(t, claimed.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
		claimed, err = c.Claim(ctx, b.candidate(t, second.Key))
		require.NoError(t, err)
		require.NoError(t, claimed.Release(ctx))
		_, err = c.Claim(ctx, goneCandidate)
		require.ErrorIs(t, err, ErrTaskGone)
		_, err = c.Claim(ctx, b.candidate(t, second.Key))
		require.NoError(t, err)
		_, err = c.FailOwn(ctx, "restarted")
		require.NoError(t, err)

		if b.clientset != nil {
			recordWatches(ctx, t, b, calls)
		}
	})

	ctx := logging.TestContext(t.Context())
	reviewed := &workerCalls{calls: map[string]bool{}}
	kubeClient := kubefake.NewClientset()
	kubeClient.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		review := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
		attrs := review.Spec.ResourceAttributes
		assert.Equal(t, workflow.Group, attrs.Group)
		assert.Equal(t, testNamespace, attrs.Namespace)
		reviewed.record(attrs.Verb, attrs.Resource)
		review.Status.Allowed = true
		return true, review, nil
	})
	require.NoError(t, CheckAccess(ctx, kubeClient, testWorker("worker-a")))
	assert.Equal(t, calls.sorted(), reviewed.sorted(), "CheckAccess checks every verb the worker uses")

	data, err := os.ReadFile("../../manifests/quick-start/base/hot-worker-role.yaml")
	require.NoError(t, err)
	var role rbacv1.Role
	require.NoError(t, yaml.Unmarshal(data, &role))
	// Nothing more: the worker's account is often the workflow's own.
	var granted []string
	for _, r := range role.Rules {
		assert.Equal(t, []string{workflow.Group}, r.APIGroups)
		assert.Empty(t, r.ResourceNames)
		for _, resource := range r.Resources {
			for _, verb := range r.Verbs {
				granted = append(granted, verb+" "+resource)
			}
		}
	}
	slices.Sort(granted)
	assert.Equal(t, calls.sorted(), granted)
}

// recordWatches records the calls of the worker's Watches on the
// Kubernetes backend: the list and watch of its metadata informers, and a
// claim's live read of its result, which a cached result of the claim's
// name with another UID makes.
func recordWatches(ctx context.Context, t *testing.T, b *backend, calls *workerCalls) {
	t.Helper()
	worker := testWorker("worker-a")
	scheme := metadatafake.NewTestScheme()
	require.NoError(t, metav1.AddMetaToScheme(scheme))
	older := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskResultKind},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "node-9",
			UID:       "older-uid",
			Labels:    map[string]string{common.LabelKeyPoolHash: worker.PoolHash},
		},
	}
	metadataClient := metadatafake.NewSimpleMetadataClient(scheme, older)
	metadataClient.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		calls.record(action.GetVerb(), action.GetResource().Resource)
		return false, nil, nil
	})
	metadataClient.PrependWatchReactor("*", func(action k8stesting.Action) (bool, watch.Interface, error) {
		calls.record(action.GetVerb(), action.GetResource().Resource)
		return false, nil, nil
	})
	watches := NewWatches(b.clientset, metadataClient, worker)
	watches.Run(ctx)
	require.True(t, cache.WaitForCacheSync(ctx.Done(), watches.taskSets.HasSynced, watches.results.HasSynced))
	lost := make(chan StopReason, 1)
	unwatch := watches.Watch(ctx, Claim{Key: Key{Namespace: testNamespace, NodeID: "node-9"}, UID: "newer-uid"}, func(r StopReason) { lost <- r })
	defer unwatch()
	select {
	case r := <-lost:
		assert.Equal(t, StopLost, r, "the live read found the claim's result gone")
	case <-time.After(5 * time.Second):
		t.Fatal("the claim's result wasn't read live")
	}
	require.Eventually(t, func() bool {
		got := calls.sorted()
		return slices.Contains(got, "watch "+ResourceTaskSets) && slices.Contains(got, "watch "+ResourceResults)
	}, 5*time.Second, 10*time.Millisecond, "the informers watch")
}
