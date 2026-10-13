package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowActionStatusFulfilled(t *testing.T) {
	assert.False(t, WorkflowActionStatus{}.Fulfilled())
	assert.False(t, WorkflowActionStatus{Phase: WorkflowActionPending}.Fulfilled())
	assert.True(t, WorkflowActionStatus{Phase: WorkflowActionSucceeded}.Fulfilled())
	assert.True(t, WorkflowActionStatus{Phase: WorkflowActionFailed}.Fulfilled())
}

func TestWorkflowActionUnmarshal(t *testing.T) {
	a := MustUnmarshalWorkflowAction(`
apiVersion: argoproj.io/v1alpha1
kind: WorkflowAction
metadata:
  generateName: my-wf-resume-
spec:
  workflowRef:
    name: my-wf
  action: Resume
  resume:
    nodeFieldSelector: displayName=approve
    outputParameters:
      approved: "true"
`)
	assert.Equal(t, ActionTypeResume, a.Spec.Action)
	assert.Equal(t, "my-wf", a.Spec.WorkflowRef.Name)
	assert.Equal(t, "true", a.Spec.Resume.OutputParameters["approved"])
}

func TestEffectiveShutdown(t *testing.T) {
	wf := &Workflow{}
	assert.Equal(t, ShutdownStrategyNone, wf.EffectiveShutdown())
	wf.Spec.Shutdown = ShutdownStrategyStop
	assert.Equal(t, ShutdownStrategyStop, wf.EffectiveShutdown())
	wf.Status.Shutdown = ShutdownStrategyTerminate
	assert.Equal(t, ShutdownStrategyTerminate, wf.EffectiveShutdown())
}
