package http

import (
	"bytes"
	"os"
	"regexp"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
	return buf.String()
}

func TestHTTPArtifactDriver_Load(t *testing.T) {
	driver := &HTTPArtifactDriver{}
	a := &wfv1.HTTPArtifact{
		URL: "https://github.com/argoproj/argo",
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
		output := captureOutput(func() {
			err := driver.Load(&wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
			}, "/tmp/found-with-request-headers")
			if assert.NoError(t, err) {
				_, err := os.Stat("/tmp/found-with-request-headers")
				assert.NoError(t, err)
			}
		})
		curl := "curl -fsS -L -o /tmp/found-with-request-headers https://github.com/argoproj/argo -H Accept: application/json -H Authorization: Bearer foo-bar"
		assert.Regexp(t, regexp.MustCompile(curl), output)
	})
	t.Run("NotFound", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP: &wfv1.HTTPArtifact{URL: "https://github.com/argoproj/argo/not-found"},
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

func TestHTTPArtifactDriver_Save(t *testing.T) {
	driver := &HTTPArtifactDriver{}
	assert.Error(t, driver.Save("", nil))
}
