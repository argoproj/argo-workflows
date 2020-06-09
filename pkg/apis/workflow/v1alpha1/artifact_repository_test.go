package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestArtifactRepository_IsArchiveLogs(t *testing.T) {
	assert.False(t, (&ArtifactRepository{}).IsArchiveLogs())
	assert.False(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(true)}).IsArchiveLogs())
}

func TestArtifactRepository_AsArtifactLocation(t *testing.T) {
	t.Run("ArchiveLogs", func(t *testing.T) {
		l, err := (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(true), S3: &S3ArtifactRepository{}}).AsArtifactLocation()
		if assert.NoError(t, err) {
			assert.True(t, l.IsArchiveLogs())
		}
	})
	t.Run("Artifactory", func(t *testing.T) {
		l, err := (&ArtifactRepository{Artifactory: &ArtifactoryArtifactRepository{
			RepoURL: "http://foo",
		}}).AsArtifactLocation()
		if assert.NoError(t, err) {
			if assert.NotNil(t, l.Artifactory) {
				assert.Equal(t, "http://foo/{{workflow.name}}/{{pod.name}}", l.Artifactory.URL)
			}
		}
	})
	t.Run("GCS", func(t *testing.T) {
		l, err := (&ArtifactRepository{GCS: &GCSArtifactRepository{}}).AsArtifactLocation()
		if assert.NoError(t, err) {
			if assert.NotNil(t, l.GCS) {
				assert.Equal(t, "{{workflow.name}}/{{pod.name}}", l.GCS.Key)
			}
		}
	})
	t.Run("HDFS", func(t *testing.T) {
		l, err := (&ArtifactRepository{HDFS: &HDFSArtifactRepository{}}).AsArtifactLocation()
		if assert.NoError(t, err) {
			if assert.NotNil(t, l.HDFS) {
				assert.Empty(t, l.HDFS.Path)
			}
		}
	})
	t.Run("OSS", func(t *testing.T) {
		l, err := (&ArtifactRepository{OSS: &OSSArtifactRepository{}}).AsArtifactLocation()
		if assert.NoError(t, err) {
			if assert.NotNil(t, l.OSS) {
				assert.Equal(t, "{{workflow.name}}/{{pod.name}}", l.OSS.Key)
			}
		}
	})
	t.Run("S3", func(t *testing.T) {
		l, err := (&ArtifactRepository{S3: &S3ArtifactRepository{}}).AsArtifactLocation()
		if assert.NoError(t, err) {
			assert.NotNil(t, l.S3)
		}
	})
}
