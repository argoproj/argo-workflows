package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestMetaNamespaceLabelIndex(t *testing.T) {
	assert.Equal(t, "my-ns/my-label", MetaNamespaceLabelIndex("my-ns", "my-label"))
}

func TestMetaNamespaceLabelIndexFunc(t *testing.T) {
	t.Run("NoLabel", func(t *testing.T) {
		values, err := MetaNamespaceLabelIndexFunc("my-label")(&wfv1.Workflow{})
		assert.NoError(t, err)
		assert.Empty(t, values)
	})
	t.Run("Labelled", func(t *testing.T) {
		values, err := MetaNamespaceLabelIndexFunc("my-label")(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Labels:    map[string]string{"my-label": "my-value"},
			},
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, values, []string{"my-ns/my-value"})
	})
	t.Run("Labelled No Namespace", func(t *testing.T) {
		values, err := MetaLabelIndexFunc("my-label")(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Labels:    map[string]string{"my-label": "my-value"},
			},
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, values, []string{"my-value"})
	})
}
