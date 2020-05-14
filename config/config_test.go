package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestArtifactRepository_IsArchiveLogs(t *testing.T) {
	assert.False(t, (&ArtifactRepository{}).IsArchiveLogs())
	assert.False(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactRepository{ArchiveLogs: pointer.BoolPtr(true)}).IsArchiveLogs())
}

func TestArtifactRepository_MergeInto(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		(&ArtifactRepository{}).MergeInto(&wfv1.Artifact{})
	})
	t.Run("S3", func(t *testing.T) {
		b := &wfv1.Artifact{ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{}}}
		(&ArtifactRepository{S3: &S3ArtifactRepository{S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint"}}}).MergeInto(b)
		assert.Equal(t, "my-endpoint", b.S3.Endpoint)
	})
	t.Run("GCS", func(t *testing.T) {
		b := &wfv1.Artifact{ArtifactLocation: wfv1.ArtifactLocation{GCS: &wfv1.GCSArtifact{}}}
		(&ArtifactRepository{GCS: &GCSArtifactRepository{GCSBucket: wfv1.GCSBucket{Bucket: "my-bucket¬"}}}).MergeInto(b)
		assert.Equal(t, "my-bucket", b.GCS.Bucket)
	})
	t.Run("GCS", func(t *testing.T) {
		b := &wfv1.Artifact{ArtifactLocation: wfv1.ArtifactLocation{GCS: &wfv1.GCSArtifact{}}}
		(&ArtifactRepository{GCS: &GCSArtifactRepository{GCSBucket: wfv1.GCSBucket{Bucket: "my-bucket¬"}}}).MergeInto(b)
		assert.Equal(t, "my-bucket", b.GCS.Bucket)
	})
}
