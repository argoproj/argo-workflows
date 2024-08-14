package labels

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestLabel(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo")
		require.Empty(t, obj.Labels)
	})
	t.Run("One", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "bar")
		require.Len(t, obj.Labels, 1)
		require.Equal(t, "bar", obj.Labels["foo"])
	})
	t.Run("Two", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "bar", "baz")
		require.Len(t, obj.Labels, 1)
		require.Equal(t, "bar", obj.Labels["foo"])
	})
}

func TestUnLabel(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		UnLabel(obj, "foo")
		require.Empty(t, obj.Labels)
	})
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}
		UnLabel(obj, "foo")
		require.Empty(t, obj.Labels)
	})
	t.Run("One", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": ""}}}
		UnLabel(obj, "foo")
		require.Empty(t, obj.Labels)
	})
	t.Run("Two", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"bar": ""}}}
		UnLabel(obj, "foo")
		require.Len(t, obj.Labels, 1)
		require.Contains(t, obj.Labels, "bar")
	})
}
