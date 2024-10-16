package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuthString(t *testing.T) {
	t.Setenv("ARGO_TOKEN", "my-token")
	authString, err := GetAuthString()
	require.NoError(t, err)
	assert.Equal(t, "my-token", authString)
}

func TestNamespace(t *testing.T) {
	t.Setenv("ARGO_NAMESPACE", "my-ns")
	assert.Equal(t, "my-ns", Namespace())
}

func TestCreateOfflineClient(t *testing.T) {
	t.Run("creating an offline client with no files should not fail", func(t *testing.T) {
		Offline = true
		OfflineFiles = []string{}
		_, _, err := NewAPIClient(context.TODO())

		assert.NoError(t, err)
	})

	t.Run("creating an offline client with a non-existing file should fail", func(t *testing.T) {
		Offline = true
		OfflineFiles = []string{"non-existing-file"}
		_, _, err := NewAPIClient(context.TODO())

		assert.Error(t, err)
	})
}
