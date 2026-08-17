package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// When a DAG template declares `dag.target`, only the target tasks decide the phase of the
// DAG: "we only succeed if all the target tasks have been considered". A declared target
// may be an intermediate task whose dependents are never scheduled, so the phase must not
// be derived from the leaves of the full task graph. See #693 and #3035.
func TestDAGTargetTaskDeterminesDAGPhase(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-target-phase
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        target: build
        tasks:
          - name: build
            template: echo
          - name: publish
            template: echo
            depends: build
    - name: echo
      container:
        image: argoproj/argosay:v2
`)

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	buildNode := woc.wf.Status.Nodes.FindByDisplayName("build")
	if assert.NotNil(t, buildNode) {
		assert.Equal(t, wfv1.NodeFailed, buildNode.Phase)
	}
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("publish"), "tasks outside the target should not be scheduled")
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}
