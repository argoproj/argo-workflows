package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTL(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		ttl := TTL(-1)
		err := ttl.UnmarshalJSON([]byte(`""`))
		if assert.NoError(t, err) {
			assert.Equal(t, TTL(0), ttl)
		}
	})
	t.Run("1h", func(t *testing.T) {
		ttl := TTL(-1)
		err := ttl.UnmarshalJSON([]byte(`"1h"`))
		if assert.NoError(t, err) {
			assert.Equal(t, TTL(1*time.Hour), ttl)
		}
	})
	t.Run("1d", func(t *testing.T) {
		ttl := TTL(-1)
		err := ttl.UnmarshalJSON([]byte(`"1d"`))
		if assert.NoError(t, err) {
			assert.Equal(t, TTL(24*time.Hour), ttl)
		}
	})
}
