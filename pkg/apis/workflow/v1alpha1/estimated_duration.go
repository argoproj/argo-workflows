package v1alpha1

import "time"

// EstimatedDuration is in seconds.
type EstimatedDuration int

func (d EstimatedDuration) ToDuration() time.Duration {
	return time.Second * time.Duration(d)
}

func NewEstimatedDuration(d time.Duration) EstimatedDuration {
	return EstimatedDuration(d.Seconds())
}
