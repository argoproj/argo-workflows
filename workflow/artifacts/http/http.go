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

// ArtifactDriver is the artifact driver for a HTTP URL
type ArtifactDriver struct{}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Load download artifacts from an HTTP URL
func (h *ArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	// Download the file to a local file path
	req, err := http.NewRequest("GET", inputArtifact.HTTP.URL, nil)
	if err != nil {
		return err
	}
	for _, h := range inputArtifact.HTTP.Headers {
		req.Header.Add(h.Name, h.Value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return errors.New(errors.CodeNotFound, "no found")
		default:
			return fmt.Errorf("%s", resp.Status)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// Save upload artifacts to an HTTP URL
func (h *ArtifactDriver) Save(path string, artifact *wfv1.Artifact) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, artifact.HTTP.URL, f)
	if err != nil {
		return err
	}
	for _, h := range artifact.HTTP.Headers {
		req.Header.Add(h.Name, h.Value)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.Errorf("saving artifact to %s failed with status code %d", artifact.HTTP.URL, res.StatusCode)
	}
	return nil
}

func (h *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
