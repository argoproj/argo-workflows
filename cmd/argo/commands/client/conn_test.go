package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/logging"
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
		ctx := logging.WithLogger(context.TODO(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, _, err := NewAPIClient(ctx)

		assert.NoError(t, err)
	})

	t.Run("creating an offline client with a non-existing file should fail", func(t *testing.T) {
		Offline = true
		OfflineFiles = []string{"non-existing-file"}
		ctx := logging.WithLogger(context.TODO(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, _, err := NewAPIClient(ctx)

		assert.Error(t, err)
	})
}
