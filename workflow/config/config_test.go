package config

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
	"testing"
)

func TestArtifactRepository_IsArchiveLogs(t *testing.T) {
	assert.False(t, (&ArtifactRepository{}).IsArchiveLogs())
	assert.False(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(true)}).IsArchiveLogs())
}
