package deprecation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestUninitalized(t *testing.T) {
	metricsF = nil
	ctx := logging.TestContext(t.Context())
	Record(ctx, Undefined)
}

func TestInitalized(t *testing.T) {
	count := 0
	countUndefined := 0
	countMutex := 0
	fn := func(_ context.Context, deprecation, _ string) {
		count += 1
		if deprecation == "undefined" {
			countUndefined += 1
		}
		if deprecation == "synchronization mutex" {
			countMutex += 1
		}
	}
	Initialize(fn)
	ctx := logging.TestContext(t.Context())
	Record(ctx, Undefined)
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, countUndefined)
	assert.Equal(t, 0, countMutex)
	Record(ctx, Undefined)
	assert.Equal(t, 2, count)
	assert.Equal(t, 2, countUndefined)
	assert.Equal(t, 0, countMutex)
}
