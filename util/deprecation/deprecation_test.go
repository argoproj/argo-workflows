package deprecation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/logging"
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
		count++
		if deprecation == "undefined" {
			countUndefined++
		}
		if deprecation == "synchronization mutex" {
			countMutex++
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
