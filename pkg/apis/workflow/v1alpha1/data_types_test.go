package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetArtifactIfNeeded(t *testing.T) {
	data := &DataSource{ArtifactPaths: &ArtifactPaths{Artifact{Name: "foo"}}}
	art, needed := data.GetArtifactIfNeeded()
	require.True(t, needed)
	assert.Equal(t, "foo", art.Name)
}
