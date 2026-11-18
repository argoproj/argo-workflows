package dispatch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	kubefake "k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// watchedConsumer returns a consumer of the backend with its watch-backed
// sources: the fake's own, or for the Kubernetes backend Watches over
// informers on the fake clientset, started and synced. The fake clientset
// has no metadata client, so the informers are typed; Watches reads only
// their metadata.
func (b *backend) watchedConsumer(t *testing.T, w Worker) Consumer {
	t.Helper()
	if b.fake != nil {
		return b.fake.Consumer(w)
	}
	return NewConsumer(b.clientset, w, b.watches(t, w).Sources())
}

// watches returns Watches of w's pool on the Kubernetes backend, running
// until the test ends.
func (b *backend) watches(t *testing.T, w Worker) *Watches {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	selector := labels.SelectorFromSet(labels.Set{common.LabelKeyPoolHash: w.PoolHash}).String()
	var watching sync.WaitGroup
	watching.Add(2)
	// The fake clientset's watches don't resume from the list's resource
	// version, so anything created between list and watch would be missed:
	// wait for both watches before going on.
	listWatch := func(list func(metav1.ListOptions) (runtime.Object, error), start func(metav1.ListOptions) (watch.Interface, error)) cache.ListerWatcher {
		var once sync.Once
		// The fake clientset doesn't stream lists; this tells the reflector so.
		return cache.ToListWatcherWithWatchListSemantics(&cache.ListWatch{
			ListWithContextFunc: func(_ context.Context, options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = selector
				return list(options)
			},
			WatchFuncWithContext: func(_ context.Context, options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = selector
				wi, err := start(options)
				once.Do(watching.Done)
				return wi, err
			},
		}, b.clientset)
	}
	taskSets := b.clientset.ArgoprojV1alpha1().WorkflowTaskSets(w.Namespace)
	results := b.clientset.ArgoprojV1alpha1().WorkflowTaskResults(w.Namespace)
	watches := newWatches(kubeStore{client: b.clientset}, w,
		cache.NewSharedIndexInformer(listWatch(
			func(o metav1.ListOptions) (runtime.Object, error) { return taskSets.List(ctx, o) },
			func(o metav1.ListOptions) (watch.Interface, error) { return taskSets.Watch(ctx, o) },
		), &wfv1.WorkflowTaskSet{}, 0, cache.Indexers{}),
		cache.NewSharedIndexInformer(listWatch(
			func(o metav1.ListOptions) (runtime.Object, error) { return results.List(ctx, o) },
			func(o metav1.ListOptions) (watch.Interface, error) { return results.Watch(ctx, o) },
		), &wfv1.WorkflowTaskResult{}, 0, cache.Indexers{}),
	)
	watches.Run(ctx)
	require.True(t, cache.WaitForCacheSync(ctx.Done(), watches.taskSets.HasSynced, watches.results.HasSynced))
	watching.Wait()
	return watches
}

func TestWatchedCandidates(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())

		// None of these is a candidate.
		otherPool := testTask("other-pool")
		otherPool.PoolHash = "hash-2"
		b.enqueue(t, otherPool)
		otherNamespace := testTask("other-namespace")
		otherNamespace.Namespace = "other"
		b.enqueue(t, otherNamespace)
		b.enqueue(t, testTask("claimed"))
		b.claim(t, testWorker("worker-b"), Key{Namespace: testNamespace, NodeID: "claimed"})
		// Started after the claim, so its caches start with it.
		c := b.watchedConsumer(t, testWorker("worker-a"))

		// Buffered, so the call can return after a failed test stopped waiting.
		found := make(chan []Candidate, 1)
		go func() {
			candidates, err := c.Candidates(ctx)
			assert.NoError(t, err)
			found <- candidates
		}()
		select {
		case <-found:
			t.Fatal("Candidates returned with nothing claimable")
		case <-time.After(100 * time.Millisecond):
		}

		first := b.enqueue(t, testTask("first"))
		var got []Candidate
		select {
		case got = <-found:
		case <-time.After(5 * time.Second):
			t.Fatal("Candidates didn't return the new TaskSet")
		}
		require.Len(t, got, 1)
		assert.Equal(t, first.UID, got[0].TaskSetUID)

		second := b.enqueue(t, testTask("second"))
		require.Eventually(t, func() bool {
			all, err := c.Candidates(ctx)
			return err == nil && len(all) == 2
		}, 5*time.Second, 10*time.Millisecond)
		got, err := c.Candidates(ctx)
		require.NoError(t, err)
		assert.Equal(t, first.UID, got[0].TaskSetUID, "oldest first")
		assert.Equal(t, second.UID, got[1].TaskSetUID)
	})
}

func TestWatchedCandidatesCancelled(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b *backend) {
		w := testWorker("worker-a")
		w.PoolHash = "empty"
		c := b.watchedConsumer(t, w)
		ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
		cancel()
		_, err := c.Candidates(ctx)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// eventuallyStopped waits for the claim to stop with reason.
func eventuallyStopped(t *testing.T, h Claimed, reason StopReason) {
	t.Helper()
	select {
	case <-h.Stopped():
		assert.Equal(t, reason, h.StopReason())
	case <-time.After(5 * time.Second):
		t.Errorf("claim not stopped, want %s", reason)
	}
}

// neverStopped checks the claim isn't stopped for a while.
func neverStopped(t *testing.T, h Claimed) {
	t.Helper()
	select {
	case <-h.Stopped():
		t.Errorf("claim stopped: %s", h.StopReason())
	case <-time.After(200 * time.Millisecond):
	}
}

// watchedClaim claims the task through a watched consumer of w.
func (b *backend) watchedClaim(t *testing.T, w Worker, key Key) Claimed {
	t.Helper()
	h, err := b.watchedConsumer(t, w).Claim(logging.TestContext(t.Context()), b.candidate(t, key))
	require.NoError(t, err)
	return h
}

func TestWatchedClaimNotifications(t *testing.T) {
	t.Run("Interrupt", func(t *testing.T) {
		forEachBackend(t, func(t *testing.T, b *backend) {
			ctx := logging.TestContext(t.Context())
			task := testTask("node-1")
			ts := b.enqueue(t, task)
			h := b.watchedClaim(t, testWorker("worker-a"), task.Key)
			neverStopped(t, h)
			require.NoError(t, b.dispatcher.Interrupt(ctx, task.Key, ts.UID, h.Claim().UID))
			eventuallyStopped(t, h, StopInterrupted)
			// The worker confirms; the claim is then still its own.
			require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed}))
		})
	})
	t.Run("Deleted", func(t *testing.T) {
		forEachBackend(t, func(t *testing.T, b *backend) {
			task := testTask("node-1")
			b.enqueue(t, task)
			h := b.watchedClaim(t, testWorker("worker-a"), task.Key)
			// As garbage collection does when the workflow is deleted.
			b.deleteResult(t, task.Key)
			eventuallyStopped(t, h, StopLost)
		})
	})
	// Seen as a change to another result of the claim's name, with no
	// deletion of the claim's own.
	t.Run("Replaced", func(t *testing.T) {
		forEachBackend(t, func(t *testing.T, b *backend) {
			task := testTask("node-1")
			b.enqueue(t, task)
			h := b.watchedClaim(t, testWorker("worker-a"), task.Key)
			r, ok := b.result(t, task.Key)
			require.True(t, ok)
			replacement := r.DeepCopy()
			replacement.Labels[common.LabelKeyWorkerPodUID] = "worker-b-uid"
			require.NoError(t, b.swapResult(replacement))
			eventuallyStopped(t, h, StopLost)
		})
	})
	t.Run("ReleaseIsNotALoss", func(t *testing.T) {
		forEachBackend(t, func(t *testing.T, b *backend) {
			task := testTask("node-1")
			b.enqueue(t, task)
			h := b.watchedClaim(t, testWorker("worker-a"), task.Key)
			require.NoError(t, h.Release(logging.TestContext(t.Context())))
			neverStopped(t, h)
		})
	})
	t.Run("CompleteIsNotALoss", func(t *testing.T) {
		forEachBackend(t, func(t *testing.T, b *backend) {
			task := testTask("node-1")
			ts := b.enqueue(t, task)
			h := b.watchedClaim(t, testWorker("worker-a"), task.Key)
			require.NoError(t, h.Complete(logging.TestContext(t.Context()), wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
			// The controller acknowledges, then deletes the workflow.
			require.NoError(t, b.dispatcher.Ack(logging.TestContext(t.Context()), task.Key, ts.UID))
			b.deleteResult(t, task.Key)
			neverStopped(t, h)
		})
	})
}

// The caches lag the API server: a claim isn't lost because the cache
// hasn't seen its result yet, or still holds an older result of its name.
func TestWatchesTellLagFromLoss(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	// Informers that never run: their caches stay empty.
	idle := func(obj runtime.Object) cache.SharedIndexInformer {
		return cache.NewSharedIndexInformer(&cache.ListWatch{}, obj, 0, cache.Indexers{})
	}
	w := testWorker("worker-a")
	watches := newWatches(kubeStore{client: b.clientset}, w, idle(&wfv1.WorkflowTaskSet{}), idle(&wfv1.WorkflowTaskResult{}))
	h, err := NewConsumer(b.clientset, w, watches.Sources()).Claim(ctx, b.candidate(t, task.Key))
	require.NoError(t, err)
	neverStopped(t, h)

	older := common.NewTaskResult(testNamespace, task.ResultName(), "my-wf", "wf-uid", "")
	older.UID = "older-uid"
	watches.resultChanged(older, false)
	neverStopped(t, h)
	watches.resultChanged(older, true)
	neverStopped(t, h)

	// The live read finds the claim's result replaced.
	r, ok := b.result(t, task.Key)
	require.True(t, ok)
	replacement := r.DeepCopy()
	replacement.ResourceVersion = ""
	require.NoError(t, b.replaceResult(replacement))
	watches.resultChanged(older, false)
	eventuallyStopped(t, h, StopLost)
}

// A live read that fails says nothing about the claim, which the claim's
// next write finds out.
func TestWatchesLeaveAClaimWhoseLiveReadFails(t *testing.T) {
	b := newKubeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	idle := func(obj runtime.Object) cache.SharedIndexInformer {
		return cache.NewSharedIndexInformer(&cache.ListWatch{}, obj, 0, cache.Indexers{})
	}
	w := testWorker("worker-a")
	watches := newWatches(kubeStore{client: b.clientset}, w, idle(&wfv1.WorkflowTaskSet{}), idle(&wfv1.WorkflowTaskResult{}))
	h, err := NewConsumer(b.clientset, w, watches.Sources()).Claim(ctx, b.candidate(t, task.Key))
	require.NoError(t, err)

	var reads atomic.Int32
	b.setFault(func(op Op) error {
		if op.Verb == "get" && op.Resource == ResourceResults {
			reads.Add(1)
			return errTransient
		}
		return nil
	})
	t.Cleanup(func() { b.setFault(nil) })
	older := common.NewTaskResult(testNamespace, task.ResultName(), "my-wf", "wf-uid", "")
	older.UID = "older-uid"
	watches.resultChanged(older, false)
	require.Eventually(t, func() bool { return reads.Load() > 0 }, 5*time.Second, 10*time.Millisecond)
	neverStopped(t, h)
}

// TestWatchedWorkersClaimEachTaskOnce runs workers and tasks concurrently, as
// pool pods do.
func TestWatchedWorkersClaimEachTaskOnce(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b *backend) {
		ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
		defer cancel()
		const tasks, workers = 20, 4
		var completed atomic.Int32
		claims := make(map[string]int)
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i := range workers {
			c := b.watchedConsumer(t, testWorker(fmt.Sprintf("worker-%d", i)))
			wg.Go(func() {
				for {
					candidates, err := c.Candidates(ctx)
					if err != nil {
						return
					}
					for _, candidate := range candidates {
						h, err := c.Claim(ctx, candidate)
						if errors.Is(err, ErrClaimLost) || errors.Is(err, ErrTaskGone) {
							continue
						}
						if err != nil {
							t.Errorf("claiming %s: %v", candidate.NodeID, err)
							return
						}
						mu.Lock()
						claims[candidate.NodeID]++
						mu.Unlock()
						if assert.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded})) {
							completed.Add(1)
						}
						break
					}
				}
			})
		}
		for i := range tasks {
			b.enqueue(t, testTask(fmt.Sprintf("node-%d", i)))
		}
		require.Eventually(t, func() bool { return completed.Load() == tasks }, 20*time.Second, 10*time.Millisecond)
		cancel()
		wg.Wait()
		assert.Len(t, claims, tasks)
		for node, n := range claims {
			assert.Equal(t, 1, n, node)
		}
	})
}

// NewWatches reads only metadata, selected by the worker's pool hash.
func TestNewWatchesOverMetadata(t *testing.T) {
	taskSet := func(nodeID, hash string, claimed bool) []runtime.Object {
		m := metav1.ObjectMeta{
			Namespace:       testNamespace,
			Name:            nodeID + "-task",
			UID:             types.UID(nodeID + "-uid"),
			Labels:          map[string]string{common.LabelKeyWorkflow: "my-wf", common.LabelKeyPoolName: "my-pool", common.LabelKeyPoolHash: hash},
			OwnerReferences: []metav1.OwnerReference{common.WorkflowOwnerReference("my-wf", "wf-uid")},
		}
		objs := []runtime.Object{&metav1.PartialObjectMetadata{
			TypeMeta:   metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskSetKind},
			ObjectMeta: m,
		}}
		if claimed {
			r := m.DeepCopy()
			r.Name, r.UID = nodeID, types.UID(nodeID+"-result-uid")
			objs = append(objs, &metav1.PartialObjectMetadata{
				TypeMeta:   metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskResultKind},
				ObjectMeta: *r,
			})
		}
		return objs
	}
	scheme := metadatafake.NewTestScheme()
	require.NoError(t, metav1.AddMetaToScheme(scheme))
	client := metadatafake.NewSimpleMetadataClient(scheme, slices.Concat(
		taskSet("free", "hash-1", false),
		taskSet("claimed", "hash-1", true),
		taskSet("other-pool", "hash-2", false),
	)...)
	// The candidates filter by pool hash too, so only the requests show the
	// selector is applied.
	var mu sync.Mutex
	selectors := map[string][]string{}
	client.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if list, ok := action.(k8stesting.ListAction); ok {
			mu.Lock()
			selectors[action.GetResource().Resource] = append(selectors[action.GetResource().Resource], list.GetListRestrictions().Labels.String())
			mu.Unlock()
		}
		return false, nil, nil
	})
	client.PrependWatchReactor("*", func(action k8stesting.Action) (bool, watch.Interface, error) {
		mu.Lock()
		selectors[action.GetResource().Resource] = append(selectors[action.GetResource().Resource], action.(k8stesting.WatchAction).GetWatchRestrictions().Labels.String())
		mu.Unlock()
		return false, nil, nil
	})
	ctx := logging.TestContext(t.Context())
	watches := NewWatches(fake.NewClientset(), client, testWorker("worker-a"))
	watches.Run(ctx)
	got, err := watches.Candidates(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "free", got[0].NodeID)
	want := common.LabelKeyPoolHash + "=hash-1"
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(selectors[ResourceTaskSets]) >= 2 && len(selectors[ResourceResults]) >= 2
	}, 5*time.Second, 10*time.Millisecond, "a list and a watch of each")
	mu.Lock()
	defer mu.Unlock()
	for _, resource := range []string{ResourceTaskSets, ResourceResults} {
		for _, got := range selectors[resource] {
			assert.Equal(t, want, got, resource)
		}
	}
}

// A worker not allowed a verb it uses on its pool's TaskSets or results
// fails its access check with a message saying what to grant, naming the
// verbs, the resources and the account; a review that fails is only
// logged, so the worker goes on.
func TestCheckAccess(t *testing.T) {
	for _, tc := range []struct {
		name string
		// denied are the reviews denied, as "verb resource".
		denied []string
		// reviewErr fails every review.
		reviewErr error
		// noAccount fails the review of who the worker is.
		noAccount bool
		wantErr   string
	}{
		{name: "allowed"},
		{name: "list taskSets denied", denied: []string{"list " + ResourceTaskSets}, wantErr: "isn't allowed to list workflowtasksets in namespace argo"},
		{
			name:    "writes denied",
			denied:  []string{"create " + ResourceResults, "patch " + ResourceResults, "watch " + ResourceTaskSets},
			wantErr: "isn't allowed to watch workflowtasksets; create, patch workflowtaskresults in namespace argo",
		},
		{name: "account unknown", denied: []string{"get " + ResourceTaskSets}, noAccount: true, wantErr: "to the service account the worker runs as, which"},
		{name: "review fails", reviewErr: apierr.NewServiceUnavailable("down")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := kubefake.NewClientset()
			client.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
				if tc.reviewErr != nil {
					return true, nil, tc.reviewErr
				}
				review := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
				attrs := review.Spec.ResourceAttributes
				assert.Equal(t, testNamespace, attrs.Namespace)
				assert.Equal(t, workflow.Group, attrs.Group)
				review.Status.Allowed = !slices.Contains(tc.denied, attrs.Verb+" "+attrs.Resource)
				return true, review, nil
			})
			client.PrependReactor("create", "selfsubjectreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
				if tc.noAccount {
					return true, nil, apierr.NewNotFound(authenticationv1.Resource("selfsubjectreviews"), "")
				}
				review := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.SelfSubjectReview)
				review.Status.UserInfo.Username = "system:serviceaccount:argo:runner"
				return true, review, nil
			})
			err := CheckAccess(logging.TestContext(t.Context()), client, testWorker("worker-a"))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Contains(t, err.Error(), "hot-worker Role")
			if !tc.noAccount {
				assert.Contains(t, err.Error(), "(system:serviceaccount:argo:runner)", "names the account, from the API server")
			}
		})
	}
}
