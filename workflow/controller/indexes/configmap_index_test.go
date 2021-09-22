package indexes

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigMapIndexFunc(t *testing.T) {
	t.Run("NoLabel", func(t *testing.T) {
		values, err := ConfigMapIndexFunc(&corev1.ConfigMap{})
		assert.NoError(t, err)
		assert.Equal(t, []string{""}, values)
	})
	t.Run("HasLabel", func(t *testing.T) {
		values, err := ConfigMapIndexFunc(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{ConfigMapTypeLabel: "cache"}},
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, values, []string{"cache"})
	})
}
