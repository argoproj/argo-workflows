package webhdfs

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

type WebhdfsOperation string

const (
	OPEN_OP   WebhdfsOperation = "OPEN"
	CREATE_OP WebhdfsOperation = "CREATE"
)

// to be able to mock the http client in unit tests
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ArtifactDriver is the artifact driver for an webHDFS endpoint
type ArtifactDriver struct {
	// webhdfs endpoint to do the HTTP requests against
	Endpoint string
	// custom headers
	Headers []wfv1.Header
	// the client for doing the HTTP requests
	Client HttpClient
}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Load downloads an artifact from a webHDFS endpoint via a webhdfs OPEN operation
// the response from the GET request typically results in a 307 Redirect
// in case that the http client does not follow this redirect automatically, we have to do so ourselves
func (webhdfs *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	url, err := buildUrl(webhdfs.Endpoint, inputArtifact.WebHDFS.Path, nil, OPEN_OP)
	if err != nil {
		return err
	}

	resp, err := webhdfs.doRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTemporaryRedirect {
		// we have been redirected and need to do a GET again on the given location
		url, err = resp.Location()
		if err != nil {
			return err
		}
		resp, err = webhdfs.doRequest(http.MethodGet, url.String(), nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not get input artifact: %s", resp.Status)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// Save pushes an artifact to a webHDFS endpoint via a webhdfs CREATE operation
// the same behavior with the redirect above applies here
func (webhdfs *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	url, err := buildUrl(webhdfs.Endpoint, outputArtifact.WebHDFS.Path, outputArtifact.WebHDFS.Overwrite, CREATE_OP)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)

	resp, err := webhdfs.doRequest(http.MethodPut, url.String(), reader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTemporaryRedirect {
		// we have been redirected and need to do a PUT again on the given location
		url, err = resp.Location()
		if err != nil {
			return err
		}
		// reset the file reader, in case it already read something
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		reader.Reset(f)
		resp, err = webhdfs.doRequest(http.MethodPut, url.String(), reader)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("could not create output artifact: %s", resp.Status)
	}
	return err
}

func (webhdfs *ArtifactDriver) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for _, h := range webhdfs.Headers {
		req.Header.Add(h.Name, h.Value)
	}
	resp, err := webhdfs.Client.Do(req)
	return resp, err
}

func (h *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type")
}
