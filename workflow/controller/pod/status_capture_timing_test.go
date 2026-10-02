package pod

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func captureTimingQueue(t *testing.T, c *Controller) *clocktesting.FakeClock {
	t.Helper()
	c.workqueue.ShutDown()
	clock := clocktesting.NewFakeClock(time.Unix(1000, 0))
	// Use production's default limiter and actual delaying queue. Only its
	// scheduling clock changes; no AddAfter shortcut makes retries immediate.
	c.workqueue = workqueue.NewTypedRateLimitingQueueWithConfig(workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{Clock: clock})
	t.Cleanup(c.workqueue.ShutDown)
	return clock
}

func captureTimingWaitScheduled(t *testing.T, clock *clocktesting.FakeClock) {
	t.Helper()
	// The delaying queue has its heartbeat ticker and a timer for the item.
	require.Eventually(t, func() bool { return clock.Waiters() >= 2 }, time.Second, time.Millisecond)
}

func captureTimingNotReady(t *testing.T, c *Controller) {
	t.Helper()
	// Give the queue goroutine a chance to expose an incorrectly early entry.
	assert.Never(t, func() bool { return c.workqueue.Len() != 0 }, 10*time.Millisecond, time.Millisecond)
}

func captureTimingVerbCounts(client *fake.Clientset) map[string]int {
	counts := map[string]int{}
	for _, action := range client.Actions() {
		counts[action.GetVerb()]++
	}
	return counts
}

func TestStatusCaptureRealQueueRetryBounds(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, offloaded := range []bool{false, true} {
		for _, failure := range []string{"forbidden", "conflict"} {
			t.Run(fmt.Sprintf("offloaded=%v/%s", offloaded, failure), func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, pod := captureFixture()
				client := fake.NewClientset(pod)
				c := identityTestController(t, client)
				clock := captureTimingQueue(t, c)
				broken := true
				workflowReads, offloadReads := 0, 0
				c.lookupWorkflow = func(_ context.Context, namespace, name string, hydrate bool) (*wfv1.Workflow, error) {
					workflowReads++
					assert.Equal(t, pod.Namespace, namespace)
					assert.Equal(t, wf.Name, name)
					workflowCopy := wf.DeepCopy()
					if offloaded && !hydrate {
						workflowCopy.Status.Nodes = nil
						workflowCopy.Status.OffloadNodeStatusVersion = "api-version"
						return workflowCopy, nil
					}
					if broken {
						if failure == "forbidden" {
							return nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, name, fmt.Errorf("denied"))
						}
						return nil, apierr.NewConflict(schema.GroupResource{Resource: "workflows"}, name, fmt.Errorf("changed"))
					}
					if hydrate {
						offloadReads++
					}
					return workflowCopy, nil
				}
				key := newPodCleanupKeyWithUID(pod.Namespace, pod.Name, removeFinalizer, string(pod.UID))
				for range 100 {
					c.workqueue.Add(key)
				}
				require.Equal(t, 1, c.workqueue.Len(), "duplicate delivery shares one queued UID action")
				require.True(t, c.processNextPodCleanupItem(ctx))
				assert.Equal(t, map[string]int{"get": 1}, captureTimingVerbCounts(client))
				delay := podCleanupRetryDelay
				if failure == "conflict" {
					delay = 5 * time.Millisecond // production default per-item first retry
					assert.Equal(t, 1, c.workqueue.NumRequeues(key))
				} else {
					assert.Zero(t, c.workqueue.NumRequeues(key), "fixed delayed retry does not consume exponential backoff")
				}
				captureTimingWaitScheduled(t, clock)
				clock.Step(delay - time.Nanosecond)
				captureTimingNotReady(t, c)
				assert.Equal(t, map[string]int{"get": 1}, captureTimingVerbCounts(client))
				broken = false // no Pod update, informer event, or restart
				clock.Step(time.Nanosecond)
				require.Eventually(t, func() bool { return c.workqueue.Len() == 1 }, time.Second, time.Millisecond)
				require.True(t, c.processNextPodCleanupItem(ctx))
				assert.Equal(t, map[string]int{"get": 2, "patch": 1}, captureTimingVerbCounts(client))
				wantWorkflowReads := 2
				if offloaded {
					wantWorkflowReads = 4
					assert.Equal(t, 1, offloadReads)
				} else {
					assert.Zero(t, offloadReads)
				}
				assert.Equal(t, wantWorkflowReads, workflowReads)
				assert.Zero(t, c.workqueue.NumRequeues(key), "success forgets the actual UID-bound retry state")
				assert.Zero(t, c.workqueue.Len())

				// Reordered duplicate cleanup after success performs a fresh Pod
				// GET, but no repeat Workflow reads or mutation once barrier is gone.
				for range 100 {
					c.workqueue.Add(key)
				}
				require.Equal(t, 1, c.workqueue.Len())
				require.True(t, c.processNextPodCleanupItem(ctx))
				assert.Equal(t, map[string]int{"get": 3, "patch": 1}, captureTimingVerbCounts(client))
				assert.Equal(t, wantWorkflowReads, workflowReads)
				clock.Step(time.Hour)
				captureTimingNotReady(t, c)
				assert.Equal(t, map[string]int{"get": 3, "patch": 1}, captureTimingVerbCounts(client))
				current, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, []string{"example.com/keep"}, current.Finalizers)
			})
		}
	}
}

func TestStatusCaptureRealQueueRecoveryDeleteDelay(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	wf, pod := captureFixture()
	pod.Finalizers = []string{common.FinalizerPodStatus}
	wf.Spec.PodGC = &wfv1.PodGC{Strategy: wfv1.PodGCOnWorkflowCompletion, DeleteDelayDuration: "5s"}
	client := fake.NewClientset(pod)
	c := identityTestController(t, client)
	clock := captureTimingQueue(t, c)
	workflowReads := 0
	c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
		workflowReads++
		return wf.DeepCopy(), nil
	}
	recoveryKey := newPodCleanupKeyWithUID(pod.Namespace, pod.Name, reconcilePodCleanup, string(pod.UID))
	for range 100 {
		c.workqueue.Add(recoveryKey)
	}
	require.Equal(t, 1, c.workqueue.Len())
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Equal(t, map[string]int{"get": 1}, captureTimingVerbCounts(client))
	captureTimingWaitScheduled(t, clock)
	clock.Step(2 * time.Second)
	captureTimingNotReady(t, c)
	// Duplicate recovery may rediscover the same deletion. The real delaying
	// queue retains the earlier deadline; it must neither run now nor reset it.
	for range 100 {
		c.queuePodForCleanupAfter(ctx, pod.Namespace, pod.Name, deletePod, 5*time.Second, string(pod.UID))
	}
	captureTimingNotReady(t, c)
	clock.Step(3*time.Second - time.Nanosecond)
	captureTimingNotReady(t, c)
	assert.Equal(t, map[string]int{"get": 1}, captureTimingVerbCounts(client))
	clock.Step(time.Nanosecond)
	require.Eventually(t, func() bool { return c.workqueue.Len() == 1 }, time.Second, time.Millisecond)
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Equal(t, map[string]int{"get": 2, "patch": 1, "delete": 1}, captureTimingVerbCounts(client))
	assert.Equal(t, 2, workflowReads, "recovery and final mutation independently read Workflow")
	clock.Step(time.Hour)
	captureTimingNotReady(t, c)
	assert.Equal(t, map[string]int{"get": 2, "patch": 1, "delete": 1}, captureTimingVerbCounts(client))
	assert.Zero(t, c.workqueue.NumRequeues(newPodCleanupKeyWithUID(pod.Namespace, pod.Name, deletePod, string(pod.UID))))
}
