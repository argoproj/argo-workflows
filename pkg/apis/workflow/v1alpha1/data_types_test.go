package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPodPolicy(t *testing.T) {
	data := &Data{}
	assert.False(t, data.UsePod())

	data = &Data{Source: DataSource{ArtifactPaths: &ArtifactPaths{Artifact{}}}}
	assert.True(t, data.UsePod())
}

func TestGetArtifactIfAny(t *testing.T) {
	data := &Data{}
	assert.Nil(t, data.GetArtifactIfAny())

	data = &Data{Source: DataSource{ArtifactPaths: &ArtifactPaths{Artifact{Name: "foo"}}}}
	art := data.GetArtifactIfAny()
	if assert.NotNil(t, art) {
		assert.Equal(t, "foo", art.Name)
	}
}
