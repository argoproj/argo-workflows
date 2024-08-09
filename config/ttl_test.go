package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTL(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		ttl := TTL(-1)
		err := ttl.UnmarshalJSON([]byte(`""`))
		require.NoError(t, err)
		assert.Equal(t, TTL(0), ttl)
	})
	t.Run("1h", func(t *testing.T) {
		ttl := TTL(-1)
		err := ttl.UnmarshalJSON([]byte(`"1h"`))
		require.NoError(t, err)
		assert.Equal(t, TTL(1*time.Hour), ttl)
	})
	t.Run("1d", func(t *testing.T) {
		ttl := TTL(-1)
		err := ttl.UnmarshalJSON([]byte(`"1d"`))
		require.NoError(t, err)
		assert.Equal(t, TTL(24*time.Hour), ttl)
	})
}
