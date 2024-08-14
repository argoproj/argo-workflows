package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEstimatedDuration(t *testing.T) {
	duration := NewEstimatedDuration(time.Minute)
	require.Equal(t, EstimatedDuration(60), duration)
	require.Equal(t, time.Duration(time.Minute), duration.ToDuration())
}
