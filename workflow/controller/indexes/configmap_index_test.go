package indexes

import (
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapIndexFunc(t *testing.T) {
	t.Run("NoLabel", func(t *testing.T) {
		values, err := ConfigMapIndexFunc(&corev1.ConfigMap{})
		assert.NoError(t, err)
		assert.Equal(t, []string{""}, values)
	})
	t.Run("HasLabel", func(t *testing.T) {
		values, err := ConfigMapIndexFunc(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{common.LabelKeyConfigMapType: common.LabelValueCacheTypeConfigMap}},
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, values, []string{common.LabelValueCacheTypeConfigMap})
	})
}
