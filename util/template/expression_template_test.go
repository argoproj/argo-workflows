package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_hasRetries(t *testing.T) {
	t.Run("hasRetiresInExpression", func(t *testing.T) {
		assert.True(t, hasRetries("retries"))
		assert.True(t, hasRetries("retries + 1"))
		assert.True(t, hasRetries("sprig(retries)"))
		assert.True(t, hasRetries("sprig(retries + 1) * 64"))
		assert.False(t, hasRetries("foo"))
		assert.False(t, hasRetries("retriesCustom + 1"))
	})
}
