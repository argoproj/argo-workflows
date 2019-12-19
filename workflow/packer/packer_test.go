package packer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestDecompressWorkflow(t *testing.T) {
	defer func() { maxWorkflowSize = defaultMaxWorkflowSize }()
	maxWorkflowSize = 300

	t.Run("SmallWorkflow", func(t *testing.T) {
		wf, err := CompressWorkflow(&wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}},
			},
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
		}
		wf, err = DecompressWorkflow(wf)
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
		}
	})
	t.Run("LargeWorkflow", func(t *testing.T) {
		wf, err := CompressWorkflow(&wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}},
			},
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.Empty(t, wf.Status.Nodes)
			assert.NotEmpty(t, wf.Status.CompressedNodes)
		}
		wf, err = DecompressWorkflow(wf)
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
		}
	})
	t.Run("TooLargeToCompressWorkflow", func(t *testing.T) {
		wf, err := CompressWorkflow(&wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}, "baz": wfv1.NodeStatus{}, "qux": wfv1.NodeStatus{}},
			},
		})
		if assert.Error(t, err) {
			assert.True(t, IsTooLargeError(err))
			assert.Nil(t, wf)
		}
	})
}
