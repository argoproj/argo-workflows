package labels

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestLabel(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "")
		assert.Empty(t, obj.Labels)
	})
	t.Run("One", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "bar")
		assert.Len(t, obj.Labels, 1)
		assert.Equal(t, "bar", obj.Labels["foo"])
	})
}

func TestUnLabel(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo", "")
		assert.Empty(t, obj.Labels)
	})
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}
		Label(obj, "foo", "")
		assert.Empty(t, obj.Labels)
	})
	t.Run("One", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": ""}}}
		Label(obj, "foo", "")
		assert.Empty(t, obj.Labels)
	})
	t.Run("Two", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"bar": ""}}}
		Label(obj, "foo", "")
		assert.Len(t, obj.Labels, 1)
		assert.Contains(t, obj.Labels, "bar")
	})
}
