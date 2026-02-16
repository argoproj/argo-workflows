package rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	ss, err := String(10)
	require.NoError(t, err)
	assert.Len(t, ss, 10)
	ss, err = String(5)
	require.NoError(t, err)
	assert.Len(t, ss, 5)
}
