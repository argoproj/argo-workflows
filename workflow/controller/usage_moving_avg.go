package controller

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// https://en.wikipedia.org/wiki/Moving_average#Cumulative_moving_average
type usageMovingAvg struct {
	xs corev1.ResourceList
	n  int
}

func (a usageMovingAvg) String() string {
	var parts []string
	for n, q := range a.xs {
		parts = append(parts, fmt.Sprintf("%s=%v", n, q.String()))
	}
	return strings.Join(parts, ",")
}

func (a usageMovingAvg) Add(usage corev1.ResourceList) usageMovingAvg {
	a1 := usageMovingAvg{
		xs: corev1.ResourceList{},
		n:  a.n + 1,
	}
	for n1, x1 := range usage {
		an0 := a.xs[n1]
		c := resource.NewQuantity((an0.Value()*int64(a1.n-1)+x1.Value())/int64(a1.n), x1.Format)
		a1.xs[n1] = *c
	}
	return a1
}
