package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestHTTPArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactDriver{Client: http.DefaultClient}
	a := &wfv1.HTTPArtifact{
		URL: "https://github.com/argoproj/argo-workflows",
	}
	t.Run("Found", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
		}, "/tmp/found")
		require.NoError(t, err)
		_, err = os.Stat("/tmp/found")
		require.NoError(t, err)
	})
	t.Run("FoundWithRequestHeaders", func(t *testing.T) {
		h1 := wfv1.Header{Name: "Accept", Value: "application/json"}
		h2 := wfv1.Header{Name: "Authorization", Value: "Bearer foo-bar"}
		a.Headers = []wfv1.Header{h1, h2}
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
		}, "/tmp/found-with-request-headers")
		require.NoError(t, err)
		_, err = os.Stat("/tmp/found-with-request-headers")
		require.NoError(t, err)
		assert.FileExists(t, "/tmp/found-with-request-headers")
	})
	t.Run("NotFound", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP: &wfv1.HTTPArtifact{URL: "https://github.com/argoproj/argo-workflows/not-found"},
			},
		}, "/tmp/not-found")
		require.Error(t, err)
		argoError, ok := err.(errors.ArgoError)
		if assert.True(t, ok) {
			assert.Equal(t, errors.CodeNotFound, argoError.Code())
		}
	})
}

func TestArtifactoryArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactDriver{Client: http.DefaultClient}
	t.Run("NotFound", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo-workflows/not-found"},
			},
		}, "/tmp/not-found")
		require.Error(t, err)
		argoError, ok := err.(errors.ArgoError)
		if assert.True(t, ok) {
			assert.Equal(t, errors.CodeNotFound, argoError.Code())
		}
	})
	t.Run("Found", func(t *testing.T) {
		err := driver.Load(&wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo-workflows"},
			},
		}, "/tmp/found")
		require.NoError(t, err)
		_, err = os.Stat("/tmp/found")
		require.NoError(t, err)
	})
}

func TestSaveHTTPArtifactRedirect(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "webhdfs-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir) // clean up

	tempFile := path.Join(tempDir, "tmpfile")
	content := "temporary file's content"
	if err := os.WriteFile(tempFile, []byte(content), 0o600); err != nil {
		panic(err)
	}

	firstRequest := true
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstRequest {
			// first response sends out only the 307
			w.Header().Add("Location", r.RequestURI)
			w.WriteHeader(http.StatusTemporaryRedirect)
			firstRequest = false
		} else {
			// check that content is really there
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(r.Body)
			require.NoError(t, err)
			assert.Equal(t, content, buf.String())

			w.WriteHeader(http.StatusCreated)
		}

	}))
	defer svr.Close()

	t.Run("SaveHTTPArtifactRedirect", func(t *testing.T) {
		driver := ArtifactDriver{
			Client: &http.Client{},
		}
		art := wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP: &wfv1.HTTPArtifact{
					URL: svr.URL,
				},
			},
		}
		err := driver.Save(tempFile, &art)
		require.NoError(t, err)
	})

}
