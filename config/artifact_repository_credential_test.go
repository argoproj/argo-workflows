package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestArtifactRepositoryCredential_MergeInto(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		(&ArtifactRepositoryCredential{}).MergeInto(&wfv1.Artifact{})
	})
	t.Run("S3", func(t *testing.T) {
		b := &wfv1.Artifact{ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{}}}
		(&ArtifactRepositoryCredential{S3: &wfv1.S3Bucket{Endpoint: "my-endpoint"}}).MergeInto(b)
		assert.Equal(t, "my-endpoint", b.S3.Endpoint)
	})
}

func TestArtifactRepositoryCredentials_Find(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		c, found := ArtifactRepositoryCredentials{{Name: "my-name"}}.Find(func(c ArtifactRepositoryCredential) bool { return true })
		assert.True(t, found)
		assert.Equal(t, "my-name", c.Name)
	})
	t.Run("False", func(t *testing.T) {
		c, found := ArtifactRepositoryCredentials{{Name: "my-name"}}.Find(func(c ArtifactRepositoryCredential) bool { return false })
		assert.False(t, found)
		assert.Empty(t, c)
	})
}

func TestArtifactRepositoryCredentials_Merge(t *testing.T) {
	t.Run("ErrNoCredential", func(t *testing.T) {
		_, err := ArtifactRepositoryCredentials{}.Merge(wfv1.Artifacts{{ArtifactLocation: wfv1.ArtifactLocation{CredentialName: "my-cred"}}})
		assert.Error(t, err)
	})
	t.Run("None", func(t *testing.T) {
		merged, err := ArtifactRepositoryCredentials{}.Merge(wfv1.Artifacts{{Name: "my-name"}})
		if assert.NoError(t, err) && assert.Len(t, merged, 1) {
			assert.Equal(t, "my-name", merged[0].Name)
		}
	})
	t.Run("One", func(t *testing.T) {
		merged, err := ArtifactRepositoryCredentials{{Name: "my-cred"}}.Merge(wfv1.Artifacts{{ArtifactLocation: wfv1.ArtifactLocation{CredentialName: "my-cred"}}})
		if assert.NoError(t, err) {
			assert.Len(t, merged, 1)
		}
	})
}
