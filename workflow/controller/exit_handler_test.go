package controller

import (
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestStepsOnExitTmpl(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/steps-on-exit-tmpl.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

func TestDAGOnExitTmpl(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/dag-on-exit-tmpl.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

func TestStepsOnExitTmplWithArt(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/steps-on-exit-tmpl-with-art.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	for idx, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafA") {
			node.Outputs = &wfv1.Outputs{
				Artifacts: wfv1.Artifacts{
					{
						Name: "result",
						ArtifactLocation: wfv1.ArtifactLocation{
							S3: &wfv1.S3Artifact{Key: "test"},
						},
					},
				},
			}
			woc.wf.Status.Nodes[idx] = node
			woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
		}
	}
	woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc1.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

func TestDAGOnExitTmplWithArt(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/dag-on-exit-tmpl-with-art.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	for idx, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafA") {
			node.Outputs = &wfv1.Outputs{
				Artifacts: wfv1.Artifacts{
					{
						Name: "result",
						ArtifactLocation: wfv1.ArtifactLocation{
							S3: &wfv1.S3Artifact{Key: "test"},
						},
					},
				},
			}
			woc.wf.Status.Nodes[idx] = node
			woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
		}
	}
	woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc1.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

func TestStepsTmplOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/steps-tmpl-on-exit.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc1.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc1.wf.Status.Phase)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}

	assert.True(t, onExitNodeIsPresent)
	makePodsPhase(ctx, woc1, apiv1.PodSucceeded)
	woc2 := newWorkflowOperationCtx(ctx, woc1.wf, controller)
	woc2.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc2.wf.Status.Phase)
	makePodsPhase(ctx, woc2, apiv1.PodSucceeded)
	for idx, node := range woc2.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafB") {
			node.Outputs = &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{
						Name:  "result",
						Value: wfv1.AnyStringPtr("Welcome"),
					},
				},
			}
			woc2.wf.Status.Nodes[idx] = node
			woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
		}
	}

	woc3 := newWorkflowOperationCtx(ctx, woc2.wf, controller)
	woc3.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc3.wf.Status.Phase)
	onExitNodeIsPresent = false
	for _, node := range woc3.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

func TestDAGOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/dag-on-exit.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc1 := newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc1.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc1.wf.Status.Phase)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)

	makePodsPhase(ctx, woc1, apiv1.PodSucceeded)
	woc2 := newWorkflowOperationCtx(ctx, woc1.wf, controller)
	woc2.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc2.wf.Status.Phase)
	makePodsPhase(ctx, woc2, apiv1.PodSucceeded)
	for idx, node := range woc2.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafB") {
			node.Outputs = &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{
						Name:  "result",
						Value: wfv1.AnyStringPtr("Welcome"),
					},
				},
			}
			woc2.wf.Status.Nodes[idx] = node
			woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
		}
	}
	woc3 := newWorkflowOperationCtx(ctx, woc2.wf, controller)
	woc3.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc3.wf.Status.Phase)
	onExitNodeIsPresent = false
	for _, node := range woc3.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

func TestDagOnExitAndRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/dag-on-exit-and-retry-strategy.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestWorkflowOnExitHttpReconciliation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/workflow-on-exit-http-reconciliation.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	taskSets, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, taskSets.Items)
	woc.operate(ctx)

	assert.Len(t, woc.wf.Status.Nodes, 2)
	taskSets, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, taskSets.Items, 1)
}

func TestWorkflowOnExitStepsHttpReconciliation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/workflow-on-exit-steps-http-reconciliation.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	taskSets, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, taskSets.Items)

	woc.operate(ctx)

	assert.Len(t, woc.wf.Status.Nodes, 4)
	taskSets, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, taskSets.Items, 1)
}

func TestWorkflowOnExitWorkflowStatus(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/workflow-on-exit-workflow-status.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	taskSets, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, taskSets.Items)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestStepsTemplateOnExitStatusArgument(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/exit_handler/steps-template-on-exit-status-argument.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	var hasExitNode bool
	var exitNodeName string

	for _, node := range woc.wf.Status.Nodes {
		if node.IsExitNode() {
			hasExitNode = true
			exitNodeName = node.DisplayName
			break
		}
	}

	assert.True(t, hasExitNode)
	assert.NotEmpty(t, exitNodeName)

	hookNode := woc.wf.Status.Nodes.FindByDisplayName(exitNodeName)

	require.NotNil(t, hookNode)
	assert.NotNil(t, hookNode.Inputs)
	require.Len(t, hookNode.Inputs.Parameters, 1)
	assert.NotNil(t, hookNode.Inputs.Parameters[0].Value)
	assert.Equal(t, hookNode.Inputs.Parameters[0].Value.String(), string(apiv1.PodFailed))
}
