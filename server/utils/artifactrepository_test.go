package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	armocks "github.com/argoproj/argo-workflows/v4/workflow/artifactrepositories/mocks"
)

func TestResolveArtifactLocation(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("returns location when default repo exists", func(t *testing.T) {
		repositories := armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
			S3: &wfv1.S3ArtifactRepository{
				S3Bucket: wfv1.S3Bucket{
					Endpoint: "my-endpoint",
					Bucket:   "my-bucket",
				},
			},
		})

		location, err := ResolveArtifactLocation(ctx, repositories, nil, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, location)
		require.NotNil(t, location.S3)
		assert.Equal(t, "my-endpoint", location.S3.Endpoint)
		assert.Equal(t, "my-bucket", location.S3.Bucket)
	})

	t.Run("returns nil location when no default repo is configured", func(t *testing.T) {
		repositories := armocks.DummyArtifactRepositories(nil)

		location, err := ResolveArtifactLocation(ctx, repositories, nil, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, location)
	})
}
