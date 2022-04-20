package http

import (
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
	var req *http.Request
	var url string
	if inputArtifact.Artifactory != nil && inputArtifact.HTTP == nil {
		url = inputArtifact.Artifactory.URL
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(h.Username, h.Password)
	} else {
		url = inputArtifact.HTTP.URL
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		for _, h := range inputArtifact.HTTP.Headers {
			req.Header.Add(h.Name, h.Value)
		}
		if h.Username != "" && h.Password != "" {
			req.SetBasicAuth(h.Username, h.Password)
		}
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode == 404 {
		return errors.New(errors.CodeNotFound, res.Status)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("loading file from %s failed with reason:%s", url, res.Status)
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
	var req *http.Request
	var url string
	if outputArtifact.Artifactory != nil && outputArtifact.HTTP == nil {
		url = outputArtifact.Artifactory.URL
		req, err = http.NewRequest(http.MethodPut, url, f)
		if err != nil {
			return err
		}
		req.SetBasicAuth(h.Username, h.Password)
	} else {
		url = outputArtifact.HTTP.URL
		req, err = http.NewRequest(http.MethodPut, url, f)
		if err != nil {
			return err
		}
		for _, h := range outputArtifact.HTTP.Headers {
			req.Header.Add(h.Name, h.Value)
		}
		if h.Username != "" && h.Password != "" {
			req.SetBasicAuth(h.Username, h.Password)
		}
	}
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("saving file %s to %s failed with reason:%s", path, url, res.Status)
	}
	return nil
}

func (h *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
