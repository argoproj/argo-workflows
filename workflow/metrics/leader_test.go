package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestIsLeader(t *testing.T) {
	_, te, err := createTestMetrics(
		&Config{},
		Callbacks{
			IsLeader: func() bool {
				return true
			},
		})

	require.NoError(t, err)
	assert.NotNil(t, te)
	attribs := attribute.NewSet()
	val, err := te.GetInt64GaugeValue(`is_leader`, &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestNotLeader(t *testing.T) {
	_, te, err := createTestMetrics(
		&Config{},
		Callbacks{
			IsLeader: func() bool {
				return false
			},
		})
	require.NoError(t, err)
	assert.NotNil(t, te)
	attribs := attribute.NewSet()
	val, err := te.GetInt64GaugeValue(`is_leader`, &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)
}
