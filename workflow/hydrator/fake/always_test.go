package fake

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestAlways(t *testing.T) {
	h := Always
	wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}}}}
	ctx := logging.TestContext(t.Context())
	t.Run("Dehydrate", func(t *testing.T) {
		err := h.Dehydrate(ctx, wf)
		require.NoError(t, err)
		assert.False(t, h.IsHydrated(wf))
		assert.Empty(t, wf.Status.Nodes)
		assert.NotEmpty(t, wf.Status.OffloadNodeStatusVersion)
	})
	t.Run("Hydrate", func(t *testing.T) {
		err := h.Hydrate(ctx, wf)
		require.NoError(t, err)
		assert.True(t, h.IsHydrated(wf))
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.OffloadNodeStatusVersion)
	})
	t.Run("HydrateWithNodes", func(t *testing.T) {
		err := h.Dehydrate(ctx, wf)
		require.NoError(t, err)
		h.HydrateWithNodes(wf, wfv1.Nodes{"foo": wfv1.NodeStatus{}})
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.OffloadNodeStatusVersion)
	})
}
