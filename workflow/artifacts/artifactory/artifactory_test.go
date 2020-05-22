package artifactory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestArtifactoryArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactoryArtifactDriver{}
	t.Run("NotFound", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo/not-found"},
			},
		}, "/tmp/not-found")
		if assert.Error(t, err) {
			argoError, ok := err.(errors.ArgoError)
			if assert.True(t, ok) {
				assert.Equal(t, errors.CodeNotFound, argoError.Code())
			}
		}
	})
	t.Run("Found", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo"},
			},
		}, "/tmp/found")
		if assert.NoError(t, err) {
			_, err := os.Stat("/tmp/found")
			assert.NoError(t, err)
		}
	})
}
