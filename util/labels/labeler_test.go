package labels

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestLabel(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo")
		assert.Empty(t, obj.Labels)
	})
	t.Run("One", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "bar")
		assert.Len(t, obj.Labels, 1)
		assert.Equal(t, "bar", obj.Labels["foo"])
	})
	t.Run("Two", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "bar", "baz")
		assert.Len(t, obj.Labels, 1)
		assert.Equal(t, "bar", obj.Labels["foo"])
	})
}
