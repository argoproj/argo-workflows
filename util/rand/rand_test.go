package rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandString(t *testing.T) {
	ss, err := RandString(10)
	require.NoError(t, err)
	assert.Len(t, ss, 10)
	ss, err = RandString(5)
	require.NoError(t, err)
	assert.Len(t, ss, 5)
}
