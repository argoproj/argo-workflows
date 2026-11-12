package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/sync"
)

const wfWithWorkflowLevelSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-wf-level-sem
  namespace: default
spec:
  entrypoint: whalesay
  synchronization:
    semaphores:
      - configMapKeyRef:
          key: workflow
          name: my-config
  templates:
    - name: whalesay
      container:
        args:
          - "hello world"
        command:
          - cowsay
        image: "docker/whalesay:latest"
`

// The semaphore's configmap does not exist in these tests, so TryAcquire
// returns an error. A transient error must leave the workflow/node pending
// and requeue; only a non-transient error may fail it.

func TestSyncErrorWorkflowLevel(t *testing.T) {
	operateWorkflow := func(t *testing.T) *wfOperationCtx {
		t.Helper()
		cancel, controller := newController(logging.TestContext(t.Context()))
		defer cancel()
		ctx := logging.TestContext(t.Context())
		controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
		}, workflowExistenceFunc, false)

		wf := wfv1.MustUnmarshalWorkflow(wfWithWorkflowLevelSemaphore)
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		return woc
	}

	t.Run("NonTransientErrorFailsWorkflow", func(t *testing.T) {
		woc := operateWorkflow(t)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "Failed to acquire the synchronization lock")
	})

	t.Run("TransientErrorLeavesWorkflowPending", func(t *testing.T) {
		t.Setenv("TRANSIENT_ERROR_PATTERN", "failed to initialize semaphore")
		woc := operateWorkflow(t)
		assert.Equal(t, wfv1.WorkflowPending, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "Waiting to acquire the synchronization lock")
	})
}

func TestSyncTransientErrorNodeLevel(t *testing.T) {
	t.Setenv("TRANSIENT_ERROR_PATTERN", "failed to initialize semaphore")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)

	wf := wfv1.MustUnmarshalWorkflow(wfWithSemaphore)
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.NotEqual(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	require.NotEmpty(t, woc.wf.Status.Nodes)
	for _, node := range woc.wf.Status.Nodes {
		assert.Equal(t, wfv1.NodePending, node.Phase, "a transient sync error must leave the node pending, not errored")
	}
}
