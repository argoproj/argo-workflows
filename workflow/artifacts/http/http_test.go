package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestHTTPArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactDriver{Client: http.DefaultClient}
	a := &wfv1.HTTPArtifact{
		URL: "https://github.com/argoproj/argo-workflows",
	}
	tempDir := t.TempDir()

	t.Run("Found", func(t *testing.T) {
		tempFile := filepath.Join(tempDir, "found")
		ctx := logging.TestContext(t.Context())
		err := driver.Load(ctx, &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
		}, tempFile)
		require.NoError(t, err)
		_, err = os.Stat(tempFile)
		require.NoError(t, err)
	})
	t.Run("FoundWithRequestHeaders", func(t *testing.T) {
		tempFile := filepath.Join(tempDir, "found-with-request-headers")
		h1 := wfv1.Header{Name: "Accept", Value: "application/json"}
		h2 := wfv1.Header{Name: "Authorization", Value: "Bearer foo-bar"}
		a.Headers = []wfv1.Header{h1, h2}
		ctx := logging.TestContext(t.Context())
		err := driver.Load(ctx, &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: a},
		}, tempFile)
		require.NoError(t, err)
		_, err = os.Stat(tempFile)
		require.NoError(t, err)
		assert.FileExists(t, tempFile)
	})
	t.Run("NotFound", func(t *testing.T) {
		tempFile := filepath.Join(tempDir, "not-found")
		ctx := logging.TestContext(t.Context())
		err := driver.Load(ctx, &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP: &wfv1.HTTPArtifact{URL: "https://github.com/argoproj/argo-workflows/not-found"},
			},
		}, tempFile)
		require.Error(t, err)
		var argoError argoerrors.ArgoError
		require.ErrorAs(t, err, &argoError)
		assert.Equal(t, argoerrors.CodeNotFound, argoError.Code())
	})
}

func TestArtifactoryArtifactDriver_Load(t *testing.T) {
	driver := &ArtifactDriver{Client: http.DefaultClient}
	tempDir := t.TempDir()

	t.Run("NotFound", func(t *testing.T) {
		tempFile := filepath.Join(tempDir, "not-found")
		ctx := logging.TestContext(t.Context())
		err := driver.Load(ctx, &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo-workflows/not-found"},
			},
		}, tempFile)
		require.Error(t, err)
		var argoError argoerrors.ArgoError
		require.ErrorAs(t, err, &argoError)
		assert.Equal(t, argoerrors.CodeNotFound, argoError.Code())
	})
	t.Run("Found", func(t *testing.T) {
		tempFile := filepath.Join(tempDir, "found")
		ctx := logging.TestContext(t.Context())
		err := driver.Load(ctx, &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://github.com/argoproj/argo-workflows"},
			},
		}, tempFile)
		require.NoError(t, err)
		_, err = os.Stat(tempFile)
		require.NoError(t, err)
	})
}

func TestSaveHTTPArtifactRedirect(t *testing.T) {
	tempDir := t.TempDir()

	tempFile := filepath.Join(tempDir, "tmpfile")
	content := "temporary file's content"
	err := os.WriteFile(tempFile, []byte(content), 0o600)
	require.NoError(t, err)

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
			if assert.NoError(t, err) {
				assert.Equal(t, content, buf.String())
			}

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
		ctx := logging.TestContext(t.Context())
		err := driver.Save(ctx, tempFile, &art)
		require.NoError(t, err)
	})
}

func TestSaveStream(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	content := "streamed artifact content"

	t.Run("streams the reader body to an HTTP destination", func(t *testing.T) {
		var received string
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(r.Body)
			assert.NoError(t, err)
			received = buf.String()
			w.WriteHeader(http.StatusCreated)
		}))
		defer svr.Close()

		driver := &ArtifactDriver{Client: svr.Client()}
		art := &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: svr.URL}},
		}
		err := driver.SaveStream(ctx, strings.NewReader(content), art)
		require.NoError(t, err)
		assert.Equal(t, content, received)
	})

	t.Run("returns an error on a non-2xx response", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer svr.Close()

		driver := &ArtifactDriver{Client: svr.Client()}
		art := &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: svr.URL}},
		}
		err := driver.SaveStream(ctx, strings.NewReader(content), art)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "saving stream")
	})

	t.Run("follows a 307 redirect when SaveStreamViaFile is set", func(t *testing.T) {
		var received string
		firstRequest := true
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if firstRequest {
				w.Header().Add("Location", r.RequestURI)
				w.WriteHeader(http.StatusTemporaryRedirect)
				firstRequest = false
				return
			}
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(r.Body)
			assert.NoError(t, err)
			received = buf.String()
			w.WriteHeader(http.StatusCreated)
		}))
		defer svr.Close()

		driver := &ArtifactDriver{Client: svr.Client()}
		art := &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP: &wfv1.HTTPArtifact{URL: svr.URL, SaveStreamViaFile: true},
			},
		}
		err := driver.SaveStream(ctx, strings.NewReader(content), art)
		require.NoError(t, err)
		assert.Equal(t, content, received)
	})
}

func TestSaveStreamNilArtifactLocation(t *testing.T) {
	driver := &ArtifactDriver{Client: http.DefaultClient}
	ctx := logging.TestContext(t.Context())

	t.Run("nil HTTP and nil Artifactory", func(t *testing.T) {
		art := &wfv1.Artifact{}
		err := driver.SaveStream(ctx, bytes.NewReader([]byte("content")), art)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Either Artifactory or HTTP artifact needs to be configured")
	})

	t.Run("both HTTP and Artifactory set", func(t *testing.T) {
		art := &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP:        &wfv1.HTTPArtifact{URL: "https://example.com/artifact"},
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://example.com/artifactory"},
			},
		}
		err := driver.SaveStream(ctx, bytes.NewReader([]byte("content")), art)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Either Artifactory or HTTP artifact needs to be configured")
	})
}

// TestSaveInvalidArtifactLocation pins the behavior Save gained by sharing
// buildPutRequest with SaveStream: an ambiguous (both set) or missing location
// is now rejected with an error, where it previously dereferenced nil HTTP or
// silently preferred HTTP over Artifactory.
func TestSaveInvalidArtifactLocation(t *testing.T) {
	driver := &ArtifactDriver{Client: http.DefaultClient}
	ctx := logging.TestContext(t.Context())

	tempFile := filepath.Join(t.TempDir(), "artifact")
	require.NoError(t, os.WriteFile(tempFile, []byte("content"), 0o600))

	t.Run("nil HTTP and nil Artifactory", func(t *testing.T) {
		err := driver.Save(ctx, tempFile, &wfv1.Artifact{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Either Artifactory or HTTP artifact needs to be configured")
	})

	t.Run("both HTTP and Artifactory set", func(t *testing.T) {
		art := &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{
				HTTP:        &wfv1.HTTPArtifact{URL: "https://example.com/artifact"},
				Artifactory: &wfv1.ArtifactoryArtifact{URL: "https://example.com/artifactory"},
			},
		}
		err := driver.Save(ctx, tempFile, art)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Either Artifactory or HTTP artifact needs to be configured")
	})
}
