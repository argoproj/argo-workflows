package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	sqldbmocks "github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
)

func TestCapturedPodCompletedOffloadFailureRetriesWithoutPublishing(t *testing.T) {
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "true")
	for _, phase := range []wfv1.WorkflowPhase{wfv1.WorkflowSucceeded, wfv1.WorkflowFailed, wfv1.WorkflowError} {
		t.Run(string(phase), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureWriterFixture()
			cancel, controller := newController(ctx, wf)
			defer cancel()
			require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
			clock := clocktesting.NewFakeClock(time.Unix(1000, 0))
			controller.wfQueue.ShutDown()
			controller.wfQueue = workqueue.NewTypedRateLimitingQueueWithConfig(&fixedItemIntervalRateLimiter{requeueTime: controller.requeueTime}, workqueue.TypedRateLimitingQueueConfig[string]{Clock: clock})
			t.Cleanup(controller.wfQueue.ShutDown)
			broken := true
			var saved wfv1.Nodes
			repo := &sqldbmocks.OffloadNodeStatusRepo{}
			repo.On("Save", string(wf.UID), wf.Namespace, mock.Anything).
				Run(func(args mock.Arguments) { saved = args.Get(2).(wfv1.Nodes).DeepCopy() }).
				Return("committed-version", func(string, string, wfv1.Nodes) error {
					if broken {
						return fmt.Errorf("offload INSERT denied")
					}
					return nil
				})
			controller.hydrator = hydrator.New(repo)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			old := wf.Status.Nodes[wf.Name]
			selected := woc.assessNodeStatus(ctx, pod, &old)
			require.NotNil(t, selected)
			selected.Phase = wfv1.NodePhase(phase)
			woc.wf.Status.Nodes[wf.Name] = *selected
			woc.wf.Status.Phase = phase
			woc.wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			woc.updated = true
			client := controller.wfclientset.(*fakewfclientset.Clientset)
			client.ClearActions()
			require.NotPanics(t, func() { woc.persistUpdates(ctx) })
			assert.True(t, woc.reapplyFailed)
			assert.Equal(t, phase, woc.wf.Status.Phase, "keep the selected terminal outcome")
			assert.Equal(t, string(pod.UID), woc.wf.Status.Nodes[wf.Name].CapturedPodUID)
			for _, action := range client.Actions() {
				assert.NotEqual(t, "update", action.GetVerb(), "no Workflow reference may be published after failed node save")
			}
			assert.Zero(t, controller.PodController.TestingQueueNumRequeues(fmt.Sprintf("%s/%s/labelPodCompleted/%s", pod.Namespace, pod.Name, pod.UID)))
			persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, wfv1.WorkflowRunning, persisted.Status.Phase)
			assert.Empty(t, persisted.Status.OffloadNodeStatusVersion)
			assert.Empty(t, persisted.Status.Nodes[wf.Name].CapturedPodUID)

			// The actual delaying queue must make the Workflow eligible again
			// without an event, and never before the configured retry interval.
			require.Eventually(t, func() bool { return clock.Waiters() >= 2 }, time.Second, time.Millisecond)
			clock.Step(30*time.Second - time.Nanosecond)
			assert.Never(t, func() bool { return controller.wfQueue.Len() != 0 }, 10*time.Millisecond, time.Millisecond)
			clock.Step(time.Nanosecond)
			require.Eventually(t, func() bool { return controller.wfQueue.Len() == 1 }, time.Second, time.Millisecond)
			key, quit := controller.wfQueue.Get()
			require.False(t, quit)
			controller.wfQueue.Done(key)
			expectedKey, err := cache.MetaNamespaceKeyFunc(wf)
			require.NoError(t, err)
			assert.Equal(t, expectedKey, key)

			broken = false
			retry := newWorkflowOperationCtx(ctx, persisted, controller)
			retry.wf.Status.Nodes[wf.Name] = *selected
			retry.wf.Status.Phase = phase
			retry.wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			retry.updated = true
			retry.persistUpdates(ctx)
			require.False(t, retry.reapplyFailed)
			persisted, err = client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, phase, persisted.Status.Phase)
			assert.Equal(t, "committed-version", persisted.Status.OffloadNodeStatusVersion)
			assert.Empty(t, persisted.Status.Nodes)
			repo.On("Get", string(wf.UID), "committed-version").Return(saved.DeepCopy(), nil).Once()
			read, err := controller.lookupWorkflowForPodCleanup(ctx, wf.Namespace, wf.Name, true)
			require.NoError(t, err)
			assert.Equal(t, selected.Phase, read.Status.Nodes[wf.Name].Phase)
			assert.Equal(t, string(pod.UID), read.Status.Nodes[wf.Name].CapturedPodUID)
			assert.Positive(t, controller.PodController.TestingQueueNumRequeues(fmt.Sprintf("%s/%s/labelPodCompleted/%s", pod.Namespace, pod.Name, pod.UID)))
			repo.AssertNumberOfCalls(t, "Save", 2)
			repo.AssertExpectations(t)
		})
	}
}
