package webhdfs

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type mockBody struct {
	*bytes.Buffer // already implements the io.Reader interface
}

func (cb *mockBody) Close() (err error) {
	return // just to mock the io.ReadCloser interface
}

type mockHttpClient struct {
	// mock http client for inspecting the made http requests from the webhdfs artifact driver
	// and simulating the redirect behavior
	mockedErr              error
	mockedRedirectError    error
	mockedResponse         *http.Response
	mockedRedirectResponse *http.Response
	mandatoryHeaders       []wfv1.Header
	expectedUrl            string
	expectedRedirectUrl    string
	t                      *testing.T
}

// if the mockedRedirectErr/Response are set, than the client mock will simulate a single redirect, returning them
// in the next request, the mock client will then return the mockedResponse/Err
// if the redirect values are unset, it immediately returns the mockedResponse/Err (simulating that no redirect happens)
func (c *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	// check for mandatory headers
	for _, header := range c.mandatoryHeaders {
		assert.Equal(c.t, header.Value, req.Header.Get(header.Name), "header mismatch!")
	}

	// check that Do is called with correct URLs
	if c.expectedRedirectUrl != "" {
		assert.Equal(c.t, c.expectedRedirectUrl, req.URL.String(), "redirectUrl mismatch!")
		c.expectedRedirectUrl = ""
	} else {
		assert.Equal(c.t, c.expectedUrl, req.URL.String(), "url mismatch!")
	}

	if c.mockedRedirectResponse != nil || c.mockedRedirectError != nil {
		// set to nil, so we only redirect once
		tmpResp, tmpErr := c.mockedRedirectResponse, c.mockedRedirectError
		c.mockedRedirectResponse = nil
		c.mockedRedirectError = nil
		return tmpResp, tmpErr
	}
	return c.mockedResponse, c.mockedErr
}

func TestLoadWebhdfsArtifact(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "webhdfs-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir) // clean up

	tests := map[string]struct {
		client    HttpClient
		endpoint  string
		path      string
		headers   []wfv1.Header
		localPath string
		done      bool
		errMsg    string
	}{
		"SuccessNoRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusOK,
					Body:       &mockBody{bytes.NewBufferString("Some file")},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:           t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: path.Join(tempDir, "file_no_redirect.txt"),
			errMsg:    "",
		},
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
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: path.Join(tempDir, "file_with_redirect.txt"),
			errMsg:    "",
		},
		"SuccessWithHeaders": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusOK,
					Body:       &mockBody{bytes.NewBufferString("Some file")},
				},
				mandatoryHeaders: []wfv1.Header{
					{Name: "hello", Value: "world"},
					{Name: "ciao", Value: "kakao"},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:           t,
			},
			endpoint: "https://myurl.com/webhdfs/v1",
			path:     "file.txt",
			headers: []wfv1.Header{
				{Name: "hello", Value: "world"},
				{Name: "ciao", Value: "kakao"},
			},
			localPath: path.Join(tempDir, "file_with_headers.txt"),
			errMsg:    "",
		},
		"FailNoRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       &mockBody{},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:           t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: path.Join(tempDir, "nonexisting.txt"),
			errMsg:    "could not get input artifact: 404 Not Found",
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
					Body:       &mockBody{bytes.NewBufferString("Some file")},
					Header:     map[string][]string{"Location": {"https://redirected"}},
				},
				expectedUrl:         "https://redirected",
				expectedRedirectUrl: "https://myurl.com/webhdfs/v1/file.txt?op=OPEN",
				t:                   t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: path.Join(tempDir, "nonexisting.txt"),
			errMsg:    "could not get input artifact: 500 Something went wrong",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driver := ArtifactDriver{
				Endpoint: tc.endpoint,
				Client:   tc.client,
				Headers:  tc.headers,
			}
			art := wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					WebHDFS: &wfv1.WebHDFSArtifact{
						Endpoint: tc.endpoint,
						Path:     tc.path,
						Headers:  tc.headers,
					},
				},
			}
			err := driver.Load(&art, tc.localPath)
			if err != nil {
				assert.Equal(t, tc.errMsg, err.Error())
				_, err := os.Stat(tc.localPath)
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.errMsg, "")
				_, err := os.Stat(tc.localPath)
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveWebhdfsArtifact(t *testing.T) {
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
	boolTrue, boolFalse := true, false

	tests := map[string]struct {
		client    HttpClient
		endpoint  string
		path      string
		overwrite *bool
		headers   []wfv1.Header
		localPath string
		done      bool
		errMsg    string
	}{
		"SuccessNoRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusCreated,
					Body:       &mockBody{},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
				t:           t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: tempFile,
			errMsg:    "",
		},
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
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: tempFile,
			errMsg:    "",
		},
		"SuccessWithHeaders": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusCreated,
					Body:       &mockBody{},
				},
				mandatoryHeaders: []wfv1.Header{
					{Name: "hello", Value: "world"},
					{Name: "ciao", Value: "kakao"},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
				t:           t,
			},
			endpoint: "https://myurl.com/webhdfs/v1",
			path:     "file.txt",
			headers: []wfv1.Header{
				{Name: "hello", Value: "world"},
				{Name: "ciao", Value: "kakao"},
			},
			localPath: tempFile,
			errMsg:    "",
		},
		"SuccessWithOverwriteTrue": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusCreated,
					Body:       &mockBody{},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE&overwrite=true",
				t:           t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			overwrite: &boolTrue,
			localPath: tempFile,
			errMsg:    "",
		},
		"SuccessWithOverwriteFalse": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusCreated,
					Body:       &mockBody{},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE&overwrite=false",
				t:           t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			overwrite: &boolFalse,
			localPath: tempFile,
			errMsg:    "",
		},
		"FailNoRedirect": {
			client: &mockHttpClient{
				mockedResponse: &http.Response{
					StatusCode: http.StatusBadRequest,
					Status:     "400 Bad Request",
					Body:       &mockBody{},
				},
				expectedUrl: "https://myurl.com/webhdfs/v1/file.txt?op=CREATE",
				t:           t,
			},
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: tempFile,
			errMsg:    "could not create output artifact: 400 Bad Request",
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
			endpoint:  "https://myurl.com/webhdfs/v1",
			path:      "file.txt",
			localPath: tempFile,
			errMsg:    "could not create output artifact: 500 Something went wrong",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driver := ArtifactDriver{
				Endpoint: tc.endpoint,
				Client:   tc.client,
				Headers:  tc.headers,
			}
			art := wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					WebHDFS: &wfv1.WebHDFSArtifact{
						Endpoint:  tc.endpoint,
						Path:      tc.path,
						Headers:   tc.headers,
						Overwrite: tc.overwrite,
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
