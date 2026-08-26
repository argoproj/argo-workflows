package common

import (
	"fmt"
	"math"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
)

// RetryBackoffWait returns how long to wait before the next retry attempt of
// a retry node that has already made `attempts` attempts, per the strategy's
// backoff: duration * factor^(attempts-1), capped by backoff.cap. It returns
// 0 when no backoff is configured. backoff.maxDuration is a deadline measured
// from the first attempt, not a per-attempt wait, so it is not applied here.
//
// This is the single implementation of the backoff formula; both the operator
// (which enforces the wait) and the DAG evaluator (which assesses it) use it.
func RetryBackoffWait(rs *wfv1.RetryStrategy, attempts int) (time.Duration, error) {
	if rs == nil || rs.Backoff == nil {
		return 0, nil
	}
	if rs.Backoff.Duration == "" {
		return 0, fmt.Errorf("no base duration specified for retryStrategy")
	}
	baseDuration, err := wfv1.ParseStringToDuration(rs.Backoff.Duration)
	if err != nil {
		return 0, err
	}
	timeToWait := baseDuration
	factor, err := intstr.Int32(rs.Backoff.Factor)
	if err != nil {
		return 0, err
	}
	if factor != nil && *factor > 0 {
		// timeToWait = duration * factor^retry_number; the first retry waits
		// exactly `duration`.
		multiplier := math.Pow(float64(*factor), float64(attempts-1))
		// Prevent overflow: saturate if the multiplication would exceed MaxInt64.
		if multiplier > float64(math.MaxInt64)/float64(baseDuration) {
			timeToWait = time.Duration(math.MaxInt64)
		} else {
			timeToWait = baseDuration * time.Duration(multiplier)
		}
	}
	if rs.Backoff.Cap != "" {
		capDuration, err := wfv1.ParseStringToDuration(rs.Backoff.Cap)
		if err != nil {
			return 0, err
		}
		if timeToWait > capDuration {
			timeToWait = capDuration
		}
	}
	return timeToWait, nil
}
