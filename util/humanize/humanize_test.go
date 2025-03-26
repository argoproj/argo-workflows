package humanize

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRelativeDurationShort tests RelativeDurationShort
func TestRelativeDurationShort(t *testing.T) {
	start := time.Now()
	end := start
	assert.Equal(t, "0s", RelativeDurationShort(start, end))

	start = time.Now().Add(-1 * time.Hour)
	end = time.Time{}
	assert.Equal(t, "1h", RelativeDurationShort(start, end))

	start = time.Now().Add(-1 * (time.Hour + 30*time.Minute))
	end = time.Time{}
	assert.Equal(t, "1h", RelativeDurationShort(start, end))

	start = time.Time{}
	end = time.Time{}
	assert.Equal(t, "0s", RelativeDurationShort(start, end))
}

// TestDuration tests Duration
func TestDuration(t *testing.T) {
	assert.Equal(t, "1 second", Duration(time.Second))
	assert.Equal(t, "1 minute 0 seconds", Duration(time.Minute))
	assert.Equal(t, "1 hour 0 minutes", Duration(time.Hour))
	assert.Equal(t, "2 hours 0 minutes", Duration(2*time.Hour))
	assert.Equal(t, "10 hours 0 minutes", Duration(10*time.Hour))
	assert.Equal(t, "1 day 0 hours", Duration(24*time.Hour))
}

// TestDuration tests Duration
func TestRelativeDuration(t *testing.T) {
	start := time.Now()
	end := start
	assert.Equal(t, "0 seconds", RelativeDuration(start, end))

	start = time.Now()
	end = start.Add(-1 * time.Second)
	assert.Equal(t, "1 second", RelativeDuration(start, end))

	start = time.Now()
	end = start.Add(-59 * time.Second)
	assert.Equal(t, "59 seconds", RelativeDuration(start, end))

	start = time.Now().Add(-90 * time.Second)
	end = time.Time{}
	assert.Equal(t, "1 minute 30 seconds", RelativeDuration(start, end))

	start = time.Now().Add(-1 * time.Hour)
	end = time.Time{}
	assert.Equal(t, "1 hour 0 minutes", RelativeDuration(start, end))

	start = time.Now().Add(-1 * (time.Hour + 30*time.Minute))
	end = time.Time{}
	assert.Equal(t, "1 hour 30 minutes", RelativeDuration(start, end))

	start = time.Time{}
	end = time.Time{}
	assert.Equal(t, "0 seconds", RelativeDuration(start, end))
}
