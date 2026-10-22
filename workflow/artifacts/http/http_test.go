package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestSaveRetry(t *testing.T) {
	t.Setenv("EXECUTOR_RETRY_BACKOFF_DURATION", "1ms")
	t.Setenv("EXECUTOR_RETRY_BACKOFF_STEPS", "3")
	content := "main container log"
	tempFile := filepath.Join(t.TempDir(), "main.log")
	require.NoError(t, os.WriteFile(tempFile, []byte(content), 0o600))

	save := func(t *testing.T, url string) error {
		t.Helper()
		driver := &ArtifactDriver{Client: &http.Client{}}
		art := &wfv1.Artifact{
			ArtifactLocation: wfv1.ArtifactLocation{Artifactory: &wfv1.ArtifactoryArtifact{URL: url}},
		}
		return driver.Save(logging.TestContext(t.Context()), tempFile, art)
	}

	tests := []struct {
		name             string
		statuses         []int // response status per attempt, the last one repeats
		transientPattern string
		wantAttempts     int32
		wantErr          string
	}{
		{name: "succeeds after 5xx responses", statuses: []int{http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusCreated}, wantAttempts: 3},
		{name: "retries 408 and 429", statuses: []int{http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusCreated}, wantAttempts: 3},
		{name: "gives up after EXECUTOR_RETRY_BACKOFF_STEPS attempts", statuses: []int{http.StatusInternalServerError}, wantAttempts: 3, wantErr: "500 Internal Server Error"},
		{name: "does not retry a 4xx", statuses: []int{http.StatusForbidden}, wantAttempts: 1, wantErr: "403 Forbidden"},
		{name: "retries a 4xx matching TRANSIENT_ERROR_PATTERN", statuses: []int{http.StatusUnauthorized, http.StatusCreated}, transientPattern: "401 Unauthorized", wantAttempts: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TRANSIENT_ERROR_PATTERN", tt.transientPattern)
			var attempts atomic.Int32
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// every attempt must re-send the whole file
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, content, string(body))
				n := int(attempts.Add(1))
				w.WriteHeader(tt.statuses[min(n, len(tt.statuses))-1])
			}))
			defer svr.Close()

			err := save(t, svr.URL)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.wantAttempts, attempts.Load())
		})
	}

	t.Run("retries a dropped connection", func(t *testing.T) {
		var attempts atomic.Int32
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) == 1 {
				conn, _, err := http.NewResponseController(w).Hijack()
				if assert.NoError(t, err) {
					_ = conn.Close()
				}
				return
			}
			w.WriteHeader(http.StatusCreated)
		}))
		defer svr.Close()

		require.NoError(t, save(t, svr.URL))
		assert.Equal(t, int32(2), attempts.Load())
	})
}
