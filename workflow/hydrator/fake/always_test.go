package fake

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestAlways(t *testing.T) {
	h := Always
	wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}}}}
	t.Run("Dehydrate", func(t *testing.T) {
		err := h.Dehydrate(wf)
		require.NoError(t, err)
		require.False(t, h.IsHydrated(wf))
		require.Empty(t, wf.Status.Nodes)
		require.NotEmpty(t, wf.Status.OffloadNodeStatusVersion)
	})
	t.Run("Hydrate", func(t *testing.T) {
		err := h.Hydrate(wf)
		require.NoError(t, err)
		require.True(t, h.IsHydrated(wf))
		require.NotEmpty(t, wf.Status.Nodes)
		require.Empty(t, wf.Status.OffloadNodeStatusVersion)
	})
	t.Run("HydrateWithNodes", func(t *testing.T) {
		err := h.Dehydrate(wf)
		require.NoError(t, err)
		h.HydrateWithNodes(wf, wfv1.Nodes{"foo": wfv1.NodeStatus{}})
		require.NotEmpty(t, wf.Status.Nodes)
		require.Empty(t, wf.Status.OffloadNodeStatusVersion)
	})
}
