package controller

import (
	"testing"
	"testing/synctest"
	"time"

	syncpkg "github.com/argoproj/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/scheme"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

const (
	outdatedRetryKey          = "default/workflow"
	outdatedRetryFloor        = 500 * time.Millisecond
	outdatedRetryRoutineDelay = 10 * time.Second
	// outdatedRetryWritten is the version the controller last wrote; the informer
	// starts two versions behind it.
	outdatedRetryWritten = "10"
	outdatedRetryBehind  = "8"
)

// newOutdatedRetryController builds a controller whose workflow informer (never
// started) holds a copy two versions older than the controller's last write.
func newOutdatedRetryController(t *testing.T, floor *time.Duration) *WorkflowController {
	t.Helper()
	controller, _ := newCompletionDelayController(t, outdatedRetryRoutineDelay, floor)
	controller.wfInformer = cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
	controller.workflowKeyLock = syncpkg.NewKeyLock()
	controller.lastWrittenVersions.versions = map[types.UID]lastWrittenVersion{}
	controller.recordWorkflowWrite(outdatedRetryWorkflow(outdatedRetryWritten))
	require.NoError(t, controller.wfInformer.GetIndexer().Add(outdatedRetryWorkflow(outdatedRetryBehind)))
	return controller
}

func outdatedRetryWorkflow(resourceVersion string) *unstructured.Unstructured {
	wf := &unstructured.Unstructured{}
	wf.SetAPIVersion(workflow.APIVersion)
	wf.SetKind(workflow.WorkflowKind)
	wf.SetNamespace("default")
	wf.SetName("workflow")
	wf.SetUID("uid")
	wf.SetResourceVersion(resourceVersion)
	return wf
}

// informerDelivers stores the new copy in the informer and runs the workflow
// update handler's enqueue for it.
func informerDelivers(t *testing.T, controller *WorkflowController, resourceVersion string) {
	t.Helper()
	wf := outdatedRetryWorkflow(resourceVersion)
	require.NoError(t, controller.wfInformer.GetIndexer().Update(wf))
	controller.enqueueWorkflowUpdate(logging.TestContext(t.Context()), outdatedRetryKey, wf)
}

// processWhenReady waits until the key is ready, checking it took want, then
// runs one worker iteration on it.
func processWhenReady(t *testing.T, controller *WorkflowController, want time.Duration) {
	t.Helper()
	assertReadyAfter(t, controller, want)
	assert.True(t, controller.processNextItem(logging.TestContext(t.Context())))
}

// assertReadyAfter checks the key becomes ready exactly want from now.
func assertReadyAfter(t *testing.T, controller *WorkflowController, want time.Duration) {
	t.Helper()
	synctest.Wait()
	time.Sleep(want - time.Nanosecond)
	synctest.Wait()
	require.Zero(t, controller.wfQueue.Len(), "ready before %s", want)
	time.Sleep(time.Nanosecond)
	synctest.Wait()
	require.Equal(t, 1, controller.wfQueue.Len(), "not ready after %s", want)
}

// TestOutdatedCompletionRetry checks that a completion reconcile skipped because the
// informer is behind runs again after minCompletionDelay once the informer
// delivers the controller's last write, and that nothing else changes timing.
func TestOutdatedCompletionRetry(t *testing.T) {
	t.Run("caught up", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryFloor)
			require.Zero(t, controller.wfQueue.Len(), "outdated reconcile is skipped")
			informerDelivers(t, controller, outdatedRetryWritten)
			assertReadyAfter(t, controller, outdatedRetryFloor)
		})
	})
	t.Run("still behind, then caught up", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryFloor)
			informerDelivers(t, controller, "9")
			time.Sleep(time.Second)
			synctest.Wait()
			require.Zero(t, controller.wfQueue.Len(), "an update older than the last write keeps waiting")
			informerDelivers(t, controller, outdatedRetryWritten)
			assertReadyAfter(t, controller, outdatedRetryFloor)
		})
	})
	t.Run("caught up while skipping", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			// The update arrived after the worker read the old copy but before it
			// recorded the skip. The handler found no skip and rate-limited the key,
			// so only the worker can re-enqueue it at the floor.
			informerDelivers(t, controller, outdatedRetryWritten)
			controller.skipOutdatedCompletion(logging.TestContext(t.Context()), outdatedRetryKey)
			assertReadyAfter(t, controller, outdatedRetryFloor)
		})
	})
	t.Run("routine reconcile while still behind", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryFloor)
			// A throttler or semaphore enqueue bypasses the rate limit; that
			// reconcile is skipped as outdated too and must keep the retry.
			controller.wfQueue.Add(outdatedRetryKey)
			synctest.Wait()
			require.Equal(t, 1, controller.wfQueue.Len())
			assert.True(t, controller.processNextItem(logging.TestContext(t.Context())))
			informerDelivers(t, controller, outdatedRetryWritten)
			assertReadyAfter(t, controller, outdatedRetryFloor)
		})
	})
	t.Run("routine event", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			controller.wfQueue.AddRateLimited(outdatedRetryKey)
			processWhenReady(t, controller, outdatedRetryRoutineDelay)
			informerDelivers(t, controller, outdatedRetryWritten)
			assertReadyAfter(t, controller, outdatedRetryRoutineDelay)
		})
	})
	t.Run("disabled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, nil)
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryRoutineDelay)
			informerDelivers(t, controller, outdatedRetryWritten)
			assertReadyAfter(t, controller, outdatedRetryRoutineDelay)
		})
	})
}

// TestOutdatedCompletionRetryRecords checks when the per-key records are dropped, so
// they cannot outlive the reconcile or workflow they belong to.
func TestOutdatedCompletionRetryRecords(t *testing.T) {
	t.Run("a reconcile takes over an earlier skip", func(t *testing.T) {
		var c outdatedCompletionRetries
		c.markSkippedAsOutdated(outdatedRetryKey)
		assert.True(t, c.startReconcile(outdatedRetryKey))
		assert.False(t, c.clearSkippedAsOutdated(outdatedRetryKey))
		assert.False(t, c.startReconcile(outdatedRetryKey))
	})
	t.Run("a reconcile consumes the completion", func(t *testing.T) {
		var c outdatedCompletionRetries
		c.markEnqueuedByCompletion(outdatedRetryKey)
		assert.True(t, c.startReconcile(outdatedRetryKey))
		assert.False(t, c.startReconcile(outdatedRetryKey))
	})
	t.Run("only one caller takes a skip", func(t *testing.T) {
		var c outdatedCompletionRetries
		c.markSkippedAsOutdated(outdatedRetryKey)
		assert.True(t, c.clearSkippedAsOutdated(outdatedRetryKey))
		assert.False(t, c.clearSkippedAsOutdated(outdatedRetryKey))
	})
	t.Run("forget drops both", func(t *testing.T) {
		var c outdatedCompletionRetries
		c.markEnqueuedByCompletion(outdatedRetryKey)
		c.markSkippedAsOutdated(outdatedRetryKey)
		c.forget(outdatedRetryKey)
		assert.False(t, c.clearSkippedAsOutdated(outdatedRetryKey))
		assert.False(t, c.startReconcile(outdatedRetryKey))
	})
}

// assertNoRecords checks that nothing is left tracked for any key.
func assertNoRecords(t *testing.T, controller *WorkflowController) {
	t.Helper()
	c := &controller.outdatedCompletionRetries
	c.mutex.Lock()
	defer c.mutex.Unlock()
	assert.Empty(t, c.enqueuedByCompletion)
	assert.Empty(t, c.skippedAsOutdated)
}

// TestOutdatedCompletionRetryNoLeak checks that a record lives only while its
// key is still queued, so the next dequeue of a key that is not skipped again
// drops it, even when the workflow is gone.
func TestOutdatedCompletionRetryNoLeak(t *testing.T) {
	t.Run("informer stays behind until the workflow is deleted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryFloor)
			// The fallback reconcile is skipped again, keeps the skip and requeues.
			processWhenReady(t, controller, outdatedRetryRoutineDelay)
			assert.Contains(t, controller.outdatedCompletionRetries.skippedAsOutdated, outdatedRetryKey)
			require.NoError(t, controller.wfInformer.GetIndexer().Delete(outdatedRetryWorkflow(outdatedRetryBehind)))
			processWhenReady(t, controller, outdatedRetryRoutineDelay)
			assertNoRecords(t, controller)
		})
	})
	t.Run("completion for a deleted workflow", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, new(outdatedRetryFloor))
			require.NoError(t, controller.wfInformer.GetIndexer().Delete(outdatedRetryWorkflow(outdatedRetryBehind)))
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryFloor)
			assertNoRecords(t, controller)
		})
	})
	t.Run("disabled records nothing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller := newOutdatedRetryController(t, nil)
			controller.enqueueNodeCompletion(logging.TestContext(t.Context()), outdatedRetryKey, podCompletionOwners)
			processWhenReady(t, controller, outdatedRetryRoutineDelay)
			informerDelivers(t, controller, outdatedRetryWritten)
			assert.Nil(t, controller.outdatedCompletionRetries.enqueuedByCompletion)
			assert.Nil(t, controller.outdatedCompletionRetries.skippedAsOutdated)
		})
	})
}

func TestMinCompletionDelay(t *testing.T) {
	controller, _ := newCompletionDelayController(t, outdatedRetryRoutineDelay, nil)
	assert.Equal(t, outdatedRetryRoutineDelay, controller.minCompletionDelay(), "unset falls back to DEFAULT_REQUEUE_TIME")
	controller.minCompletionRequeueDelay = new(outdatedRetryFloor)
	assert.Equal(t, outdatedRetryFloor, controller.minCompletionDelay())
	controller.minCompletionRequeueDelay = new(time.Minute)
	assert.Equal(t, outdatedRetryRoutineDelay, controller.minCompletionDelay(), "capped at DEFAULT_REQUEUE_TIME")
}

// TestOutdatedCompletionRetryInformer runs the real workflow informer and its
// handlers: the update that brings the informer up to the controller's last
// write retries the skipped completion, and the delete drops its record.
func TestOutdatedCompletionRetryInformer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// The routine delay is far longer than the test, so only the retry can enqueue in time.
	controller, _ := newCompletionDelayController(t, time.Hour, new(10*time.Millisecond))
	controller.workflowKeyLock = syncpkg.NewKeyLock()
	controller.lastWrittenVersions.versions = map[types.UID]lastWrittenVersion{}
	controller.recordWorkflowWrite(outdatedRetryWorkflow(outdatedRetryWritten))
	controller.throttler = controller.newThrottler()
	controller.kubeclientset = fake.NewClientset()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme.Scheme, outdatedRetryWorkflow(outdatedRetryBehind))
	workflows := dynamicClient.Resource(wfv1.WorkflowSchemaGroupVersionKind.GroupVersion().WithResource(workflow.WorkflowPlural)).Namespace("default")
	controller.wfInformer = util.NewWorkflowInformer(ctx, dynamicClient, "", 0, controller.tweakListRequestListOptions, controller.tweakWatchRequestListOptions, cache.Indexers{})
	require.NoError(t, controller.addWorkflowInformerHandlers(ctx))
	go controller.wfInformer.Run(ctx.Done())
	require.True(t, cache.WaitForCacheSync(ctx.Done(), controller.wfInformer.HasSynced))

	// Drop the enqueue from the informer's initial add.
	key, _ := controller.wfQueue.Get()
	controller.wfQueue.Done(key)

	controller.enqueueNodeCompletion(ctx, outdatedRetryKey, podCompletionOwners)
	require.True(t, controller.processNextItem(ctx))
	require.Zero(t, controller.wfQueue.Len(), "outdated reconcile is skipped")

	_, err := workflows.Update(ctx, outdatedRetryWorkflow(outdatedRetryWritten), metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return controller.wfQueue.Len() == 1 }, 5*time.Second, time.Millisecond, "the informer update retries the skipped completion")

	require.NoError(t, workflows.Delete(ctx, "workflow", metav1.DeleteOptions{}))
	require.Eventually(t, func() bool {
		controller.outdatedCompletionRetries.mutex.Lock()
		defer controller.outdatedCompletionRetries.mutex.Unlock()
		return len(controller.outdatedCompletionRetries.enqueuedByCompletion) == 0 && len(controller.outdatedCompletionRetries.skippedAsOutdated) == 0
	}, 5*time.Second, time.Millisecond, "the informer delete forgets the key")
}
