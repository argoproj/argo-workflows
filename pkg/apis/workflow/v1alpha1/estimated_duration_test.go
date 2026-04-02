package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEstimatedDuration(t *testing.T) {
	duration := NewEstimatedDuration(time.Minute)
	assert.Equal(t, EstimatedDuration(60), duration)
	assert.Equal(t, time.Minute, duration.ToDuration())
}
