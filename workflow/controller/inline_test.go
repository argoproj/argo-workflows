package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestInlineDAG(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/inline/dag.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, wfc)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestInlineSteps(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/inline/steps.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, wfc)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	node := woc.wf.Status.Nodes.FindByDisplayName("a")
	assert.Equal(t, "message", node.Inputs.Parameters[0].Name)
	assert.Equal(t, "foo", node.Inputs.Parameters[0].Value.String())
}

func TestCallTemplateWithInlineSteps(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/inline/workflow-template-with-inline-steps.yaml")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/inline/workflow-template-ref.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wftmpl)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 4)
	count := 0
	for _, pod := range pods.Items {
		nodeName := pod.Annotations["workflows.argoproj.io/node-name"]
		if strings.Contains(nodeName, "foo") {
			count++
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "foo")
		}
		if strings.Contains(nodeName, "bar") {
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "bar")
		}
	}
	assert.Equal(t, 2, count)
	for name, storedTemplate := range woc.wf.Status.StoredTemplates {
		if strings.Contains(name, "inline-a") {
			assert.Equal(t, "{{ inputs.parameters.arg }} a", storedTemplate.Container.Args[0])
		}
		if strings.Contains(name, "inline-b") {
			assert.Equal(t, "{{ inputs.parameters.arg }} b", storedTemplate.Container.Args[0])
		}
	}
}

func TestCallTemplateWithInlineDAG(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/inline/workflow-template-with-inline-dag.yaml")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/inline/workflow-template-ref.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wftmpl)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 4)
	count := 0
	for _, pod := range pods.Items {
		nodeName := pod.Annotations["workflows.argoproj.io/node-name"]
		if strings.Contains(nodeName, "foo") {
			count++
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "foo")
		}
		if strings.Contains(nodeName, "bar") {
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "bar")
		}
	}
	assert.Equal(t, 2, count)
	for name, storedTemplate := range woc.wf.Status.StoredTemplates {
		if strings.Contains(name, "inline-a") {
			assert.Equal(t, "{{ inputs.parameters.arg }} a", storedTemplate.Container.Args[0])
		}
		if strings.Contains(name, "inline-b") {
			assert.Equal(t, "{{ inputs.parameters.arg }} b", storedTemplate.Container.Args[0])
		}
	}
}
