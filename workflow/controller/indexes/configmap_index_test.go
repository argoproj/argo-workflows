package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestConfigMapIndexFunc(t *testing.T) {
	t.Run("NoLabel", func(t *testing.T) {
		values, err := ConfigMapIndexFunc(&corev1.ConfigMap{})
		require.NoError(t, err)
		assert.Empty(t, values)
	})
	t.Run("HasLabel", func(t *testing.T) {
		values, err := ConfigMapIndexFunc(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache}},
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, values, []string{common.LabelValueTypeConfigMapCache})
	})
}
