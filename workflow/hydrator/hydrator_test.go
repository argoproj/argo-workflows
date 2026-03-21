package hydrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	sqldbmocks "github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
)

func TestHydrator(t *testing.T) {
	cleanup := packer.SetMaxWorkflowSize(230)
	defer cleanup()
	ctx := logging.TestContext(t.Context())
	t.Run("Dehydrate", func(t *testing.T) {
		t.Run("Packed", func(t *testing.T) {
			hydrator := New(&sqldbmocks.OffloadNodeStatusRepo{})
			wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{CompressedNodes: "foo"}}
			err := hydrator.Dehydrate(ctx, wf)
			require.NoError(t, err)
			assert.NotEmpty(t, wf.Status.CompressedNodes)
		})
		t.Run("Offloaded", func(t *testing.T) {
			hydrator := New(&sqldbmocks.OffloadNodeStatusRepo{})
			wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{OffloadNodeStatusVersion: "foo"}}
			err := hydrator.Dehydrate(ctx, wf)
			require.NoError(t, err)
			assert.True(t, wf.Status.IsOffloadNodeStatus())
		})
		t.Run("Noop", func(t *testing.T) {
			hydrator := New(&sqldbmocks.OffloadNodeStatusRepo{})
			wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}}}}
			err := hydrator.Dehydrate(ctx, wf)
			require.NoError(t, err)
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
			assert.False(t, wf.Status.IsOffloadNodeStatus())
		})
		t.Run("Pack", func(t *testing.T) {
			hydrator := New(&sqldbmocks.OffloadNodeStatusRepo{})
			wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}}}}
			err := hydrator.Dehydrate(ctx, wf)
			require.NoError(t, err)
			assert.Empty(t, wf.Status.Nodes)
			assert.NotEmpty(t, wf.Status.CompressedNodes)
			assert.False(t, wf.Status.IsOffloadNodeStatus())
		})
		t.Run("Offload", func(t *testing.T) {
			offloadNodeStatusRepo := &sqldbmocks.OffloadNodeStatusRepo{}
			offloadNodeStatusRepo.On("Save", "my-uid", "my-ns", mock.Anything).Return("my-offload-version", nil)
			hydrator := New(offloadNodeStatusRepo)
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{UID: "my-uid", Namespace: "my-ns"},
				Spec:       wfv1.WorkflowSpec{Entrypoint: "main"},
				Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}, "baz": wfv1.NodeStatus{}, "qux": wfv1.NodeStatus{}}},
			}
			err := hydrator.Dehydrate(ctx, wf)
			require.NoError(t, err)
			assert.Empty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
			assert.True(t, wf.Status.IsOffloadNodeStatus())
			assert.Equal(t, "my-offload-version", wf.Status.OffloadNodeStatusVersion)
		})
		t.Run("WorkflowTooLargeButOffloadNotSupported", func(t *testing.T) {
			offloadNodeStatusRepo := &sqldbmocks.OffloadNodeStatusRepo{}
			offloadNodeStatusRepo.On("Save", "my-uid", "my-ns", mock.Anything).Return("my-offload-version", sqldb.ErrOffloadNotSupported)
			hydrator := New(offloadNodeStatusRepo)
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{UID: "my-uid", Namespace: "my-ns"},
				Spec:       wfv1.WorkflowSpec{Entrypoint: "main"},
				Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}, "baz": wfv1.NodeStatus{}, "qux": wfv1.NodeStatus{}}},
			}
			err := hydrator.Dehydrate(ctx, wf)
			require.Error(t, err)
		})
	})
	t.Run("Hydrate", func(t *testing.T) {
		t.Run("Offloaded", func(t *testing.T) {
			offloadNodeStatusRepo := &sqldbmocks.OffloadNodeStatusRepo{}
			offloadNodeStatusRepo.On("Get", "my-uid", "my-offload-version").Return(wfv1.Nodes{"foo": wfv1.NodeStatus{}}, nil)
			hydrator := New(offloadNodeStatusRepo)
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{UID: "my-uid"},
				Status:     wfv1.WorkflowStatus{OffloadNodeStatusVersion: "my-offload-version"},
			}
			err := hydrator.Hydrate(ctx, wf)
			require.NoError(t, err)
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
			assert.False(t, wf.Status.IsOffloadNodeStatus())
		})
		t.Run("OffloadingDisabled", func(t *testing.T) {
			offloadNodeStatusRepo := &sqldbmocks.OffloadNodeStatusRepo{}
			offloadNodeStatusRepo.On("Get", "my-uid", "my-offload-version").Return(nil, sqldb.ErrOffloadNotSupported)
			hydrator := New(offloadNodeStatusRepo)
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{UID: "my-uid"},
				Status:     wfv1.WorkflowStatus{OffloadNodeStatusVersion: "my-offload-version"},
			}
			err := hydrator.Hydrate(ctx, wf)
			require.Error(t, err)
		})
		t.Run("Packed", func(t *testing.T) {
			hydrator := New(&sqldbmocks.OffloadNodeStatusRepo{})
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{UID: "my-uid"},
				Status:     wfv1.WorkflowStatus{CompressedNodes: "H4sIAAAAAAAA/6pWSkosUrKqVspMUbJSUtJRykvMTYWwUjKLC3ISK/3gAiWVBVBWcUliUUlqimOJklVeaU6OjlJaZl5mcQZCpFZHKS0/nwbm1gICAAD//8SSRamxAAAA"},
			}
			err := hydrator.Hydrate(ctx, wf)
			require.NoError(t, err)
			assert.NotEmpty(t, wf.Status.Nodes)
			assert.Empty(t, wf.Status.CompressedNodes)
			assert.False(t, wf.Status.IsOffloadNodeStatus())
		})
		t.Run("Hydrated", func(t *testing.T) {
			hydrator := New(&sqldbmocks.OffloadNodeStatusRepo{})
			wf := &wfv1.Workflow{Status: wfv1.WorkflowStatus{}}
			err := hydrator.Hydrate(ctx, wf)
			require.NoError(t, err)
		})
	})
}
