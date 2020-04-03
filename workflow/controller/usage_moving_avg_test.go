package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_usageCapture_Add(t *testing.T) {
	assert.Equal(t, usageMovingAvg{corev1.ResourceList{}, 1}, usageMovingAvg{}.Add(corev1.ResourceList{}))

	one := *resource.NewQuantity(1, resource.DecimalSI)
	two := *resource.NewQuantity(2, resource.DecimalSI)
	six := *resource.NewQuantity(6, resource.DecimalSI)

	assert.Equal(t, usageMovingAvg{corev1.ResourceList{corev1.ResourceCPU: one}, 1},
		usageMovingAvg{}.Add(corev1.ResourceList{corev1.ResourceCPU: one}))

	// (1 * 3 + 6) / 4 =  9 / 4 = 2
	assert.Equal(t, usageMovingAvg{corev1.ResourceList{corev1.ResourceCPU: two}, 4},
		usageMovingAvg{
			xs: corev1.ResourceList{corev1.ResourceCPU: one}, n: 3,
		}.Add(corev1.ResourceList{corev1.ResourceCPU: six}))
}
