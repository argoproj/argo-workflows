package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_usageCapture_Add(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		assert.Equal(t, usageMovingAvg{corev1.ResourceList{}, 1}, usageMovingAvg{}.Add(corev1.ResourceList{}))
	})

	zero := *resource.NewQuantity(0, resource.DecimalSI)
	two := *resource.NewQuantity(200, resource.DecimalSI)
	four := *resource.NewQuantity(400, resource.DecimalSI)

	const cpu = corev1.ResourceCPU
	const memory = corev1.ResourceMemory

	t.Run("CPUAndMemory", func(t *testing.T) {
		assert.Equal(t, usageMovingAvg{corev1.ResourceList{cpu: two, memory: two}, 2},
			usageMovingAvg{}.
				Add(corev1.ResourceList{cpu: two, memory: two}).
				Add(corev1.ResourceList{cpu: two, memory: two}),
		)
	})

	t.Run("Complex", func(t *testing.T) {
		// (0+200+200+400)/400 = 2
		assert.Equal(t, usageMovingAvg{corev1.ResourceList{cpu: two}, 4},
			usageMovingAvg{}.
				Add(corev1.ResourceList{cpu: four}).
				Add(corev1.ResourceList{cpu: zero}).
				Add(corev1.ResourceList{cpu: two}).
				Add(corev1.ResourceList{cpu: two}),
		)
	})
}
