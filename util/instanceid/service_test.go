package instanceid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestLabel(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		NewService("").Label(obj)
		assert.Empty(t, obj.GetLabels())
	})
	t.Run("Add", func(t *testing.T) {
		obj := &wfv1.Workflow{}
		NewService("foo").Label(obj)
		assert.Len(t, obj.GetLabels(), 1)
		assert.Equal(t, "foo", obj.GetLabels()[common.LabelKeyControllerInstanceID])
	})
	t.Run("Remove", func(t *testing.T) {
		obj := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "bar"}}}
		NewService("").Label(obj)
		assert.Empty(t, obj.GetLabels())
	})
}

func TestWith(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		opts := &metav1.ListOptions{}
		NewService("").With(opts)
		assert.Equal(t, "!workflows.argoproj.io/controller-instanceid", opts.LabelSelector)
	})
	t.Run("EmptyExistingSelector", func(t *testing.T) {
		opts := &metav1.ListOptions{LabelSelector: "foo"}
		NewService("").With(opts)
		assert.Equal(t, "foo,!workflows.argoproj.io/controller-instanceid", opts.LabelSelector)
	})
	t.Run("ExistingSelector", func(t *testing.T) {
		opts := &metav1.ListOptions{LabelSelector: "foo"}
		NewService("foo").With(opts)
		assert.Equal(t, "foo,workflows.argoproj.io/controller-instanceid=foo", opts.LabelSelector)
	})
}

func TestValidate(t *testing.T) {
	t.Run("NoInstanceID", func(t *testing.T) {
		s := NewService("")
		assert.NoError(t, s.Validate(&wfv1.Workflow{}))
		assert.Error(t, s.Validate(&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "bar"}}}))
	})
	t.Run("InstanceID", func(t *testing.T) {
		s := NewService("foo")
		assert.Error(t, s.Validate(&wfv1.Workflow{}))
		assert.Error(t, s.Validate(&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "bar"}}}))
		assert.NoError(t, s.Validate(&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyControllerInstanceID: "foo"}}}))
	})
}

func Test_service_InstanceID(t *testing.T) {
	assert.Equal(t, "foo", NewService("foo").InstanceID())
}
