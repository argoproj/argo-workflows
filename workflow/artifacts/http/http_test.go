package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

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
	driver := &ArtifactDriver{Client: http.DefaultClient}
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

type mockBody struct {
	*bytes.Buffer // already implements the io.Reader interface
}

func (cb *mockBody) Close() (err error) {
	return // just to mock the io.ReadCloser interface
}

/*
 TESTING REDIRECT BEHAVIOR
*/
type mockHttpClient struct {
	// mock http client for inspecting the made http requests from the webhdfs artifact driver
	// and simulating the redirect behavior
	mockedResponse         *http.Response
	mockedRedirectResponse *http.Response
	expectedUrl            string
	expectedRedirectUrl    string
	t                      *testing.T
}

// than the client mock will simulate a single redirect, returning the mockedRedirectResponse on the first Do
// in the next request, the mock client will then return the mockedResponse
func (c *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	// check that Do is called with correct URLs
	if c.expectedRedirectUrl != "" {
		assert.Equal(c.t, c.expectedRedirectUrl, req.URL.String(), "redirectUrl mismatch!")
		c.expectedRedirectUrl = ""
	} else {
		assert.Equal(c.t, c.expectedUrl, req.URL.String(), "url mismatch!")
	}

	if c.mockedRedirectResponse != nil {
		// set to nil, so we only redirect once
		tmpResp := c.mockedRedirectResponse
		c.mockedRedirectResponse = nil
		return tmpResp, nil
	}
	return c.mockedResponse, nil
}

func TestLoadWebhdfsArtifactRedirect(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "webhdfs-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir) // clean up

	tests := map[string]struct {
		client          HttpClient
		url             string
		localPath       string
		done            bool
		errMsg          string
		followRedirects bool
	}{
		"SuccessWithRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusOK,
					Body:       &mockBody{bytes.NewBufferString("Some file")},
				},
				mockedRedirectResponse: &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Body:       &mockBody{},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedUrl:         "https://redirected",
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:                   t,
			},
			url:             "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
			localPath:       path.Join(tempDir, "file_with_redirect.txt"),
			followRedirects: true,
			errMsg:          "",
		},
		"FailWithRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Something went wrong",
					Body:       &mockBody{bytes.NewBufferString("Some file")},
				},
				mockedRedirectResponse: &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Body:       &mockBody{},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedUrl:         "https://redirected",
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:                   t,
			},
			url:             "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
			followRedirects: true,
			localPath:       path.Join(tempDir, "nonexisting.txt"),
			errMsg:          "loading file from https://myurl.com/webhdfs/v1/file.txt?op=OPEN failed with reason: 500 Something went wrong",
		},
		"DontFollowRedirect": {
			client: &mockHttpClient{
				mockedRedirectResponse: &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Status:     "307",
					Body:       &mockBody{},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:                   t,
			},
			url:             "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
			followRedirects: false,
			localPath:       path.Join(tempDir, "nonexisting.txt"),
			errMsg:          "loading file from https://myurl.com/webhdfs/v1/file.txt?op=OPEN failed with reason: 307",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driver := ArtifactDriver{
				Client: tc.client,
			}
			art := wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					HTTP: &wfv1.HTTPArtifact{
						URL:                      tc.url,
						FollowTemporaryRedirects: tc.followRedirects,
					},
				},
			}
			err := driver.Load(&art, tc.localPath)
			if err != nil {
				assert.Equal(t, tc.errMsg, err.Error())
			} else {
				assert.Equal(t, tc.errMsg, "")
			}
			_, err = os.Stat(tc.localPath)
			assert.NoError(t, err)
		})
	}
}

func TestSaveWebhdfsArtifactRedirect(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "webhdfs-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir) // clean up

	tempFile := path.Join(tempDir, "tmpfile")
	content := "temporary file's content"
	if err := ioutil.WriteFile(tempFile, []byte(content), 0o600); err != nil {
		panic(err)
	}

	tests := map[string]struct {
		client          HttpClient
		url             string
		localPath       string
		done            bool
		followRedirects bool
		errMsg          string
	}{
		"SuccessWithRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusCreated,
					Body:       &mockBody{},
				},
				mockedRedirectResponse: &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Body:       &mockBody{},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedUrl:         "https://redirected",
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
				t:                   t,
			},
			url:             "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
			followRedirects: true,
			localPath:       tempFile,
			errMsg:          "",
		},
		"FailWithRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Something went wrong",
					Body:       &mockBody{},
				},
				mockedRedirectResponse: &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Body:       &mockBody{},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedUrl:         "https://redirected",
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
				t:                   t,
			},
			url:             "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
			followRedirects: true,
			localPath:       tempFile,
			errMsg:          fmt.Sprintf("saving file %s to https://myurl.com/webhdfs/v1/file.txt?op=CREATE failed with reason: 500 Something went wrong", tempFile),
		},
		"DontFollowRedirects": {
			client: &mockHttpClient{
				mockedRedirectResponse: &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Status:     "307",
					Body:       &mockBody{},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
				t:                   t,
			},
			url:       "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
			localPath: tempFile,
			errMsg:    fmt.Sprintf("saving file %s to https://myurl.com/webhdfs/v1/file.txt?op=CREATE failed with reason: 307", tempFile),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driver := ArtifactDriver{
				Client: tc.client,
			}
			art := wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					HTTP: &wfv1.HTTPArtifact{
						URL:                      tc.url,
						FollowTemporaryRedirects: tc.followRedirects,
					},
				},
			}
			err := driver.Save(tc.localPath, &art)
			if err != nil {
				assert.Equal(t, tc.errMsg, err.Error())
			} else {
				assert.Equal(t, tc.errMsg, "")
			}
		})
	}
}
