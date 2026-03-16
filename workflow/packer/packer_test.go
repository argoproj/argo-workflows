package packer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestDefault(t *testing.T) {
	assert.Equal(t, 1024*1024, getMaxWorkflowSize())
}

func TestDecompressWorkflow(t *testing.T) {
	cleanup := SetMaxWorkflowSize(260)
	defer cleanup()
	ctx := logging.TestContext(t.Context())

	t.Run("SmallWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}},
			},
		}
		err := CompressWorkflowIfNeeded(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)

		err = DecompressWorkflow(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)
	})
	t.Run("LargeWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}},
			},
		}
		err := CompressWorkflowIfNeeded(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Empty(t, wf.Status.Nodes)
		assert.NotEmpty(t, wf.Status.CompressedNodes)

		err = DecompressWorkflow(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)
	})
	t.Run("TooLargeToCompressWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{Entrypoint: "main"},
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}, "baz": wfv1.NodeStatus{}, "qux": wfv1.NodeStatus{}},
			},
		}
		err := CompressWorkflowIfNeeded(ctx, wf)
		require.Error(t, err)
		assert.True(t, IsTooLargeError(err))
		// if too large, we want the original back please
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)
	})
}
