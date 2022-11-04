package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestInlineDAG(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: inline-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            inline:
              container:
                image: argoproj/argosay:v2
                args:
                  - echo
                  - "{{inputs.parameters.foo}}"
              inputs:
                parameters:
                  - name: foo
                    value: bar
`)
	cancel, wfc := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, wfc)
	woc.operate(context.Background())
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}
