package hydrator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sqldbmocks "github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/packer"
)

func TestHydrator(t *testing.T) {
	offloadNodeStatusRepo := &sqldbmocks.OffloadNodeStatusRepo{}
	hydrator := New(offloadNodeStatusRepo)

	t.Run("Noop", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}},
			},
		}
		err := hydrator.Dehydrate(wf)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
			assert.False(t, wf.Status.IsOffloadNodeStatus())
		}
	})
	t.Run("Pack", func(t *testing.T) {
		defer packer.SetMaxWorkflowSize(200)()
		wf := &wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}},
			},
		}
		err := hydrator.Dehydrate(wf)
		if assert.NoError(t, err) {
			assert.Empty(t, wf.Status.Nodes)
			assert.NotEmpty(t, wf.Status.CompressedNodes)
			assert.False(t, wf.Status.IsOffloadNodeStatus())
		}
	})

}
