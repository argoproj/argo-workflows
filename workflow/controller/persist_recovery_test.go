package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewf "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// A failed terminal write must preserve the slot and schedule recovery even if
// no more Pod events arrive. The next attempt persists the existing result.
func TestPersistFailureSchedulesRecovery(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "recover-write", Namespace: "test", Labels: map[string]string{common.LabelKeyCompleted: "false"}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning}}
	cancel, c := newController(ctx, wf)
	defer cancel()
	c.wfQueue.ShutDown()
	c.wfQueue = workqueue.NewTypedRateLimitingQueue(workqueue.NewTypedItemExponentialFailureRateLimiter[string](time.Millisecond, time.Millisecond))
	defer c.wfQueue.ShutDown()
	calls := 0
	c.wfclientset.(*fakewf.Clientset).PrependReactor("update", "workflows", func(a k8stesting.Action) (bool, runtime.Object, error) {
		calls++
		if calls == 1 {
			return true, nil, apierrors.NewInternalError(errors.New("failed calling webhook: EOF"))
		}
		return false, nil, nil
	})
	woc := newWorkflowOperationCtx(ctx, wf, c)
	woc.wf.Status.Phase = wfv1.WorkflowSucceeded
	woc.wf.Labels[common.LabelKeyCompleted] = "true"
	woc.updated = true
	woc.persistUpdates(ctx)
	require.True(t, woc.reapplyFailed)
	require.Eventually(t, func() bool { return c.wfQueue.Len() == 1 }, time.Second, time.Millisecond)
	live, err := c.wfclientset.ArgoprojV1alpha1().Workflows("test").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, wfv1.WorkflowRunning, live.Status.Phase)
	require.Empty(t, live.Labels["workflows.argoproj.io/reapply-failed"], "failure state is in memory, not a permanent label")
	again := newWorkflowOperationCtx(ctx, live, c)
	again.wf.Status.Phase = wfv1.WorkflowSucceeded
	again.wf.Labels[common.LabelKeyCompleted] = "true"
	again.updated = true
	again.persistUpdates(ctx)
	require.False(t, again.reapplyFailed)
	live, err = c.wfclientset.ArgoprojV1alpha1().Workflows("test").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, wfv1.WorkflowSucceeded, live.Status.Phase)
	require.Equal(t, 2, calls)
}
