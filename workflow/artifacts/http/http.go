package http

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// ArtifactDriver is the artifact driver for artifactory and http URLs
type ArtifactDriver struct {
	Username string
	Password string
	Client   HttpClient
}

// to be able to mock the http client in unit tests
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Load reads the artifact from the HTTP URL
func (h *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	lf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()

	var url string
	if inputArtifact.Artifactory != nil && inputArtifact.HTTP == nil {
		url = inputArtifact.Artifactory.URL
	} else {
		url = inputArtifact.HTTP.URL
	}

	res, err := h.doRequest(http.MethodGet, url, nil, inputArtifact)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 307 && inputArtifact.HTTP.FollowTemporaryRedirects {
		// we have been redirected and need to do a GET again on the given location (for webHDFS support)
		redirectUrl, err := res.Location()
		if err != nil {
			return err
		}
		res, err = h.doRequest(http.MethodGet, redirectUrl.String(), nil, inputArtifact)
		if err != nil {
			return err
		}
		defer res.Body.Close()
	}

	if res.StatusCode == 404 {
		return errors.New(errors.CodeNotFound, res.Status)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("loading file from %s failed with reason: %s", url, res.Status)
	}

	_, err = io.Copy(lf, res.Body)

	return err
}

// Save writes the artifact to the URL
func (h *ArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)

	var url string
	if outputArtifact.Artifactory != nil && outputArtifact.HTTP == nil {
		url = outputArtifact.Artifactory.URL
	} else {
		url = outputArtifact.HTTP.URL
	}

	res, err := h.doRequest(http.MethodPut, url, reader, outputArtifact)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 307 && outputArtifact.HTTP.FollowTemporaryRedirects {
		// we have been redirected and need to do a GET again on the given location (for webHDFS support)
		redirectUrl, err := res.Location()
		if err != nil {
			return err
		}
		// reset the file, in case it already read something
		_, err = f.Seek(0, io.SeekStart)
		reader.Reset(f)
		if err != nil {
			return err
		}
		res, err = h.doRequest(http.MethodPut, redirectUrl.String(), reader, outputArtifact)
		if err != nil {
			return err
		}
		defer res.Body.Close()
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("saving file %s to %s failed with reason: %s", path, url, res.Status)
	}
	return nil
}

func (h *ArtifactDriver) doRequest(method, url string, body io.Reader, artifact *wfv1.Artifact) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if artifact.Artifactory != nil && artifact.HTTP == nil {
		req.SetBasicAuth(h.Username, h.Password)
	} else {
		for _, h := range artifact.HTTP.Headers {
			req.Header.Add(h.Name, h.Value)
		}
		if h.Username != "" && h.Password != "" {
			req.SetBasicAuth(h.Username, h.Password)
		}
	}
	resp, err := h.Client.Do(req)
	return resp, err
}

func (h *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
