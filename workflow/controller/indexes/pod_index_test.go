package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMetaPodPhaseIndexFunc(t *testing.T) {
	t.Run("NoPhase", func(t *testing.T) {
		values, err := PodPhaseIndexFunc(&corev1.Pod{})
		assert.NoError(t, err)
		assert.Equal(t, []string{""}, values)
	})
	t.Run("Phase", func(t *testing.T) {
		values, err := PodPhaseIndexFunc(&corev1.Pod{
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		})
		assert.NoError(t, err)
		assert.ElementsMatch(t, values, []string{string(corev1.PodRunning)})
	})
}
