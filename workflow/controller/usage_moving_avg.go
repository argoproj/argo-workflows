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
	n  int64
}

func (a usageMovingAvg) String() string {
	var parts []string
	for n, q := range a.xs {
		parts = append(parts, fmt.Sprintf("%s=%v", n, q.String()))
	}
	return strings.Join(parts, ",")
}

func (a usageMovingAvg) Add(usage corev1.ResourceList) usageMovingAvg {
	n := a.n
	a1 := usageMovingAvg{xs: a.xs, n: n + 1}
	if a1.xs == nil {
		a1.xs = corev1.ResourceList{}
	}
	for name, value := range usage {
		// (x*n+value)/(n+1)
		tmp := a.xs[name]
		x := tmp.Value()
		q := resource.NewQuantity(x*n, tmp.Format)
		q.Add(value)
		value = *resource.NewQuantity(q.Value()/(n+1), q.Format)
		a1.xs[name] = value
	}
	return a1
}
