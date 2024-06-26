package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestIsLeader(t *testing.T) {
	_, te, err := createTestMetrics(
		&Config{},
		Callbacks{
			LeaderState: func() bool {
				return true
			},
		})
	if assert.NoError(t, err) {
		assert.NotNil(t, te)
		attribs := attribute.NewSet()
		val, err := te.GetInt64GaugeValue(`leader`, &attribs)
		if assert.NoError(t, err) {
			assert.Equal(t, int64(1), val)
		}
	}
}

func TestNotLeader(t *testing.T) {
	_, te, err := createTestMetrics(
		&Config{},
		Callbacks{
			LeaderState: func() bool {
				return false
			},
		})
	if assert.NoError(t, err) {
		assert.NotNil(t, te)
		attribs := attribute.NewSet()
		val, err := te.GetInt64GaugeValue(`leader`, &attribs)
		if assert.NoError(t, err) {
			assert.Equal(t, int64(0), val)
		}
	}
}
