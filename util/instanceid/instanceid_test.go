package instanceid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func TestLabel(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "")
		assert.Empty(t, obj.GetLabels())
	})
	t.Run("One", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		Label(obj, "foo")
		assert.Len(t, obj.GetLabels(), 1)
		assert.Equal(t, "foo", obj.GetLabels()[common.LabelKeyControllerInstanceID])
	})
}

func TestWith(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		with := With(metav1.ListOptions{}, "")
		assert.Equal(t, "!workflows.argoproj.io/controller-instanceid", with.LabelSelector)
	})
	t.Run("ExistingSelector", func(t *testing.T) {
		with := With(metav1.ListOptions{LabelSelector: "foo"}, "")
		assert.Equal(t, "foo,!workflows.argoproj.io/controller-instanceid", with.LabelSelector)
	})
}

func TestValidate(t *testing.T) {
	t.Run("NoInstanceID", func(t *testing.T) {
		assert.NoError(t, Validate(&wfv1.Workflow{}, ""))
		assert.Error(t, Validate(&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "bar"}}}, ""))
	})
	t.Run("InstanceID", func(t *testing.T) {
		assert.Error(t, Validate(&wfv1.Workflow{}, "foo"))
		assert.Error(t, Validate(&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "bar"}}}, "foo"))
		assert.NoError(t, Validate(&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "foo"}}}, "foo"))
	})
}
