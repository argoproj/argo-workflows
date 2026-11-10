package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A step exit hook whose expression can never be evaluated must fail the workflow. If the error is
// swallowed, the step group never completes and its boundary stays Running on every subsequent
// operation, so the workflow hangs forever.
// See https://github.com/argoproj/argo-workflows/issues/14031 and
// https://github.com/argoproj/argo-workflows/pull/14032.
func TestStepExitHookExpressionErrorDoesNotHangBoundary(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-hook-bad-expression
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: prepare
        template: echo
        hooks:
          exit:
            expression: steps["prepare"].outputs !
            template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
`)

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	for range 5 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		if woc.wf.Status.Fulfilled() {
			break
		}
	}

	require.Truef(t, woc.wf.Status.Fulfilled(), "workflow stuck in phase %s after the exit hook expression failed", woc.wf.Status.Phase)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "unexpected token")
}
