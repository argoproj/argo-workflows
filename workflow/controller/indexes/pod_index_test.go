package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestPhaseIndexFunc(t *testing.T) {
	t.Run("NoPhase", func(t *testing.T) {
		values, err := PodPhaseIndexFunc(&unstructured.Unstructured{})
		assert.NoError(t, err)
		assert.Equal(t, []string{""}, values)
	})
	t.Run("Phase", func(t *testing.T) {
		values, err := PodPhaseIndexFunc(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"status": map[string]interface{}{"phase": "Running"},
			},
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, values, []string{"Running"})
	})
}
