package intstr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInt(t *testing.T) {
	i, err := Int(ParsePtr("2"))
	require.NoError(t, err)
	require.Equal(t, 2, *i)

	i, err = Int(ParsePtr("-1"))
	require.NoError(t, err)
	require.Equal(t, -1, *i)

	_, err = Int(ParsePtr("{{argo.variable}}"))
	require.Error(t, err)

	i, err = Int(nil)
	require.NoError(t, err)
	require.Nil(t, i)
}

func TestInt32(t *testing.T) {
	i, err := Int32(ParsePtr("2"))
	require.NoError(t, err)
	require.Equal(t, int32(2), *i)

	i, err = Int32(ParsePtr("-1"))
	require.NoError(t, err)
	require.Equal(t, int32(-1), *i)

	_, err = Int32(ParsePtr("{{argo.variable}}"))
	require.Error(t, err)

	i, err = Int32(nil)
	require.NoError(t, err)
	require.Nil(t, i)
}

func TestInt64(t *testing.T) {
	i, err := Int64(ParsePtr("2"))
	require.NoError(t, err)
	require.Equal(t, int64(2), *i)

	i, err = Int64(ParsePtr("-1"))
	require.NoError(t, err)
	require.Equal(t, int64(-1), *i)

	_, err = Int64(ParsePtr("{{argo.variable}}"))
	require.Error(t, err)

	i, err = Int64(nil)
	require.NoError(t, err)
	require.Nil(t, i)
}

func TestIsValidIntOrArgoVariable(t *testing.T) {
	require.True(t, IsValidIntOrArgoVariable(ParsePtr("2")))
	require.True(t, IsValidIntOrArgoVariable(ParsePtr("-1")))
	require.True(t, IsValidIntOrArgoVariable(ParsePtr("{{argo.variable}}")))
	require.True(t, IsValidIntOrArgoVariable(nil))

	require.False(t, IsValidIntOrArgoVariable(ParsePtr("some-string")))
	require.False(t, IsValidIntOrArgoVariable(ParsePtr("1.5")))
}
