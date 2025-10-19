package sync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFlags(t *testing.T) {
	err := validateFlags("DATABASE", "")
	require.NoError(t, err)

	err = validateFlags("CONFIGMAP", "my-cm")
	require.NoError(t, err)

	err = validateFlags("INVALID", "")
	require.Error(t, err)
	require.Equal(t, "--type must be either 'database' or 'configmap'", err.Error())

	err = validateFlags("CONFIGMAP", "")
	require.Error(t, err)
	require.Equal(t, "--cm-name is required when type is configmap", err.Error())
}
