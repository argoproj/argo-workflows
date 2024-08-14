package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetArtifactIfNeeded(t *testing.T) {
	data := &DataSource{ArtifactPaths: &ArtifactPaths{Artifact{Name: "foo"}}}
	art, needed := data.GetArtifactIfNeeded()
	if require.True(t, needed) {
		require.Equal(t, "foo", art.Name)
	}
}
