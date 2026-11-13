package header

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/metadata"
)

func Test_nullHeaderAuth_Authorize(t *testing.T) {
	_, err := NullHeaderAuth.Authorize(metadata.MD{})
	require.Error(t, err)
}

func Test_nullHeaderAuth_IsRBACEnabled(t *testing.T) {
	assert.False(t, NullHeaderAuth.IsRBACEnabled())
}
