package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
)

func TestArtifactRepository(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var r *ArtifactRepository
		assert.Nil(t, r.Get())
		l := r.ToArtifactLocation()
		assert.Nil(t, l)
	})
	t.Run("ArchiveLogs", func(t *testing.T) {
		r := &ArtifactRepository{Artifactory: &ArtifactoryArtifactRepository{}, ArchiveLogs: ptr.To(true)}
		l := r.ToArtifactLocation()
		assert.Equal(t, ptr.To(true), l.ArchiveLogs)
	})
	t.Run("Artifactory", func(t *testing.T) {
		r := &ArtifactRepository{Artifactory: &ArtifactoryArtifactRepository{RepoURL: "http://my-repo"}}
		assert.IsType(t, &ArtifactoryArtifactRepository{}, r.Get())
		l := r.ToArtifactLocation()
		require.NotNil(t, l.Artifactory)
		assert.Equal(t, "http://my-repo/{{workflow.name}}/{{pod.name}}", l.Artifactory.URL)
	})
	t.Run("Azure", func(t *testing.T) {
		r := &ArtifactRepository{Azure: &AzureArtifactRepository{}}
		assert.IsType(t, &AzureArtifactRepository{}, r.Get())
		l := r.ToArtifactLocation()
		require.NotNil(t, l.Azure)
		assert.Equal(t, "{{workflow.name}}/{{pod.name}}", l.Azure.Blob)
	})
	t.Run("GCS", func(t *testing.T) {
		r := &ArtifactRepository{GCS: &GCSArtifactRepository{}}
		assert.IsType(t, &GCSArtifactRepository{}, r.Get())
		l := r.ToArtifactLocation()
		require.NotNil(t, l.GCS)
		assert.Equal(t, "{{workflow.name}}/{{pod.name}}", l.GCS.Key)
	})
	t.Run("HDFS", func(t *testing.T) {
		r := &ArtifactRepository{HDFS: &HDFSArtifactRepository{}}
		assert.IsType(t, &HDFSArtifactRepository{}, r.Get())
		l := r.ToArtifactLocation()
		require.NotNil(t, l.HDFS)
		assert.Equal(t, "{{workflow.name}}/{{pod.name}}", l.HDFS.Path)
	})
	t.Run("OSS", func(t *testing.T) {
		r := &ArtifactRepository{OSS: &OSSArtifactRepository{}}
		assert.IsType(t, &OSSArtifactRepository{}, r.Get())
		l := r.ToArtifactLocation()
		require.NotNil(t, l.OSS)
		assert.Equal(t, "{{workflow.name}}/{{pod.name}}", l.OSS.Key)
	})
	t.Run("S3", func(t *testing.T) {
		r := &ArtifactRepository{S3: &S3ArtifactRepository{KeyPrefix: "my-key-prefix"}}
		assert.IsType(t, &S3ArtifactRepository{}, r.Get())
		l := r.ToArtifactLocation()
		require.NotNil(t, l.S3)
		assert.Equal(t, "my-key-prefix/{{workflow.name}}/{{pod.name}}", l.S3.Key)
	})
}

func TestArtifactRepository_IsArchiveLogs(t *testing.T) {
	assert.False(t, (&ArtifactRepository{}).IsArchiveLogs())
	assert.False(t, (&ArtifactRepository{ArchiveLogs: ptr.To(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactRepository{ArchiveLogs: ptr.To(true)}).IsArchiveLogs())
}
