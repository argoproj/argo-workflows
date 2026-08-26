package common

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
)

func TestRetryBackoffWait(t *testing.T) {
	backoff := func(duration, factor, cap string) *wfv1.RetryStrategy {
		rs := &wfv1.RetryStrategy{Backoff: &wfv1.Backoff{Duration: duration, Cap: cap}}
		if factor != "" {
			rs.Backoff.Factor = intstr.ParsePtr(factor)
		}
		return rs
	}
	tests := []struct {
		name     string
		rs       *wfv1.RetryStrategy
		attempts int
		want     time.Duration
	}{
		{"no strategy", nil, 1, 0},
		{"no backoff", &wfv1.RetryStrategy{}, 1, 0},
		{"bare seconds", backoff("10", "", ""), 1, 10 * time.Second},
		{"duration string", backoff("2m", "", ""), 1, 2 * time.Minute},
		{"first retry uses base", backoff("10", "2", ""), 1, 10 * time.Second},
		{"exponential", backoff("10", "2", ""), 3, 40 * time.Second},
		{"string factor", backoff("10", "2", ""), 3, 40 * time.Second},
		{"cap", backoff("10", "2", "15"), 3, 15 * time.Second},
		{"overflow saturates", backoff("1h", "100", ""), 12, time.Duration(math.MaxInt64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RetryBackoffWait(tt.rs, tt.attempts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := RetryBackoffWait(&wfv1.RetryStrategy{Backoff: &wfv1.Backoff{}}, 1)
	assert.EqualError(t, err, "no base duration specified for retryStrategy")
	_, err = RetryBackoffWait(backoff("nonsense", "", ""), 1)
	assert.Error(t, err)
}
