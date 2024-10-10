package deprecation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUninitalized(t *testing.T) {
	metricsF = nil
	Record(context.Background(), Schedule)
}

func TestInitalized(t *testing.T) {
	count := 0
	countSchedule := 0
	countMutex := 0
	fn := func(_ context.Context, deprecation, _ string) {
		count += 1
		if deprecation == "cronworkflow schedule" {
			countSchedule += 1
		}
		if deprecation == "synchronization mutex" {
			countMutex += 1
		}
	}
	Initialize(fn)
	ctx := context.Background()
	Record(ctx, Schedule)
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, countSchedule)
	assert.Equal(t, 0, countMutex)
	Record(ctx, Mutex)
	assert.Equal(t, 2, count)
	assert.Equal(t, 1, countSchedule)
	assert.Equal(t, 1, countMutex)
}
