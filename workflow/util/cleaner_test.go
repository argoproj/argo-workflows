package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestCleanMetadata(t *testing.T) {
	CleanMetadata(nil)
}

func TestRemoveManagedFields(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		RemoveManagedFields(nil)
	})
	t.Run("Object", func(t *testing.T) {
		v := &wfv1.Workflow{}
		v.SetManagedFields([]metav1.ManagedFieldsEntry{{}})
		RemoveManagedFields(v)
		assert.Empty(t, v.GetManagedFields())
	})
	t.Run("List", func(t *testing.T) {
		v := wfv1.Workflows{{
			ObjectMeta: metav1.ObjectMeta{ManagedFields: []metav1.ManagedFieldsEntry{{}}},
		}}
		RemoveManagedFields(v)
		assert.Empty(t, v[0].GetManagedFields())
	})
}

func TestRemoveSelfLink(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		RemoveSelfLink(nil)
	})
	t.Run("Object", func(t *testing.T) {
		v := &wfv1.Workflow{}
		v.SetSelfLink("foo")
		RemoveSelfLink(v)
		assert.Empty(t, v.GetSelfLink())
	})
	t.Run("List", func(t *testing.T) {
		v := &wfv1.WorkflowList{}
		v.SetSelfLink("foo")
		RemoveSelfLink(v)
		assert.Empty(t, v.GetSelfLink())
	})
}
