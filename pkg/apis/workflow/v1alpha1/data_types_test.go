package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetArtifactIfAny(t *testing.T) {
	data := &Data{Source: DataSource{ArtifactPaths: &ArtifactPaths{Artifact{Name: "foo"}}}}
	art := data.GetArtifactIfAny()
	if assert.NotNil(t, art) {
		assert.Equal(t, "foo", art.Name)
	}
}
