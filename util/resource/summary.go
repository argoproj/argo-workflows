package resource

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type Summary struct {
	ResourceList   corev1.ResourceList
	ContainerState corev1.ContainerState
}

func (s Summary) age() time.Duration {
	if terminated := s.ContainerState.Terminated; terminated != nil {
		// A container can be terminated before it ever starts (for example when
		// image setup fails). Do not turn an unset or invalid interval into an
		// enormous or negative resource duration.
		if terminated.StartedAt.IsZero() || terminated.FinishedAt.IsZero() || terminated.FinishedAt.Time.Before(terminated.StartedAt.Time) {
			return 0
		}
		return terminated.FinishedAt.Sub(terminated.StartedAt.Time)
	}
	return 0
}

// Summaries is a map of container names to their Summary.
type Summaries map[string]Summary

func (ss Summaries) Duration() wfv1.ResourcesDuration {
	// Add container states.
	d := wfv1.ResourcesDuration{}
	for _, s := range ss {
		// age is converted to seconds, otherwise the multiplication below is very likely to overflow
		age := int64(s.age().Seconds())
		for n, q := range s.ResourceList {
			d = d.Add(wfv1.ResourcesDuration{
				n: wfv1.NewResourceDuration(time.Duration(
					q.MilliValue()*age/wfv1.ResourceQuantityDenominator(n).MilliValue(),
				) * time.Second),
			})
		}
	}
	return d
}
