package http

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestHTTPArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactDriver{}
	a := &wfv1.HTTPArtifact{
		URL: "https://github.com/argoproj/argo-workflows",
	}
	t.Run("Found", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
		}, "/tmp/found")
		if assert.NoError(t, err) {
			_, err := os.Stat("/tmp/found")
			assert.NoError(t, err)
		}
	})
	t.Run("FoundWithRequestHeaders", func(t *testing.T) {
		h1 := wfv1.Header{Name: "Accept", Value: "application/json"}
		h2 := wfv1.Header{Name: "Authorization", Value: "Bearer foo-bar"}
		a.Headers = []wfv1.Header{h1, h2}
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
		}, "/tmp/found-with-request-headers")
		if assert.NoError(t, err) {
			_, err := os.Stat("/tmp/found-with-request-headers")
			assert.NoError(t, err)
		}
		assert.FileExists(t, "/tmp/found-with-request-headers")
	})
	t.Run("NotFound", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP: &wfv1.HTTPArtifact{URL: "https://github.com/argoproj/argo-workflows/not-found"},
			},
		}, "/tmp/not-found")
		if assert.Error(t, err) {
			argoError, ok := err.(errors.ArgoError)
			if assert.True(t, ok) {
				assert.Equal(t, errors.CodeNotFound, argoError.Code())
			}
		}
	})
}

func TestArtifactoryArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactDriver{}
	t.Run("NotFound", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo-workflows/not-found"},
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
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo-workflows"},
			},
		}, "/tmp/found")
		if assert.NoError(t, err) {
			_, err := os.Stat("/tmp/found")
			assert.NoError(t, err)
		}
	})
}
