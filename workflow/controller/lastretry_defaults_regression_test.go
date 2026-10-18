package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var lastRetryNumericDefaultsWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: last-retry-defaults
spec:
  entrypoint: main
  templates:
  - name: main
    retryStrategy:
      limit: 10
    podSpecPatch: |
      containers:
        - name: main
          resources:
            limits:
              memory: "{{= (asInt(lastRetry.exitCode) + 1) * 100 }}Mi"
    container:
      image: python:alpine3.23
      command: ["python", "-c"]
      args: ["import sys; sys.exit(1)"]
`

// TestLastRetryNumericDefaultsOnFirstAttempt asserts that retries.lastRetry.exitCode and
// retries.lastRetry.duration are given a numeric "0" default on the first attempt, when there is
// no previous child node to read them from. Templates commonly feed these variables straight into
// arithmetic (asInt/sprig.int), so an empty string default makes the expression fail to resolve and
// the whole workflow errors out before the first pod is ever created.
// See #10364 and #14450.
func TestLastRetryNumericDefaultsOnFirstAttempt(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(lastRetryNumericDefaultsWorkflow)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	require.NotEqual(t, wfv1.WorkflowError, woc.wf.Status.Phase, woc.wf.Status.Message)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	assert.Equal(t, "100Mi", pods.Items[0].Spec.Containers[1].Resources.Limits.Memory().String())
}
