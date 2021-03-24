package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetArtifactIfNeeded(t *testing.T) {
	data := &DataSource{ArtifactPaths: &ArtifactPaths{Artifact{Name: "foo"}}}
	art, needed := data.GetArtifactIfNeeded()
	if assert.True(t, needed) {
		assert.Equal(t, "foo", art.Name)
	}
}
