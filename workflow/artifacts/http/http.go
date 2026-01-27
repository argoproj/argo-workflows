package http

import (
	"context"
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
	Client   *http.Client
}

var _ common.ArtifactDriver = &ArtifactDriver{}

func (h *ArtifactDriver) retrieveContent(ctx context.Context, inputArtifact *wfv1.Artifact) (http.Response, error) {
	var req *http.Request
	var url string
	var err error
	if inputArtifact.Artifactory != nil && inputArtifact.HTTP == nil {
		url = inputArtifact.Artifactory.URL
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return http.Response{}, err
		}
		req.SetBasicAuth(h.Username, h.Password)
	} else if inputArtifact.Artifactory == nil && inputArtifact.HTTP != nil {
		url = inputArtifact.HTTP.URL
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return http.Response{}, err
		}
		for _, h := range inputArtifact.HTTP.Headers {
			req.Header.Add(h.Name, h.Value)
		}
		if h.Username != "" && h.Password != "" {
			req.SetBasicAuth(h.Username, h.Password)
		}
	} else {
		return http.Response{}, errors.InternalErrorf("Either Artifactory or HTTP artifact needs to be configured")
	}

	// Note that we will close the response body in either `Load()`
	// or `ArtifactServer.returnArtifact()`, which is the caller of `OpenStream()`.
	res, err := h.Client.Do(req) //nolint:bodyclose
	if err != nil {
		return http.Response{}, err
	}
	if res.StatusCode == http.StatusNotFound {
		return http.Response{}, errors.New(errors.CodeNotFound, res.Status)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return http.Response{}, errors.InternalErrorf("loading content from %s failed with reason: %s", url, res.Status)
	}
	return *res, nil
}

// Load reads the artifact from the HTTP URL
func (h *ArtifactDriver) Load(ctx context.Context, inputArtifact *wfv1.Artifact, path string) error {
	lf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()
	res, err := h.retrieveContent(ctx, inputArtifact)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	_, err = io.Copy(lf, res.Body)
	return err
}

func (h *ArtifactDriver) OpenStream(ctx context.Context, inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	res, err := h.retrieveContent(ctx, inputArtifact)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

// Save writes the artifact to the URL
func (h *ArtifactDriver) Save(ctx context.Context, path string, outputArtifact *wfv1.Artifact) error {
	cleanPath := filepath.Clean(path)
	f, err := os.Open(cleanPath)
	if err != nil {
		return err
	}
	var req *http.Request
	var url string
	if outputArtifact.Artifactory != nil && outputArtifact.HTTP == nil {
		url = outputArtifact.Artifactory.URL
		req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, f)
		if err != nil {
			return err
		}
		req.SetBasicAuth(h.Username, h.Password)
	} else {
		url = outputArtifact.HTTP.URL
		req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, f)
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
	// we set the GetBody func of the request in order to enable following 307 POST/PUT redirects, needed e.g. for webHDFS
	req.GetBody = func() (io.ReadCloser, error) {
		return os.Open(cleanPath)
	}

	res, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("saving file %s to %s failed with reason: %s", path, url, res.Status)
	}
	return nil
}

// SaveStream saves an artifact from an io.Reader to HTTP URL
func (h *ArtifactDriver) SaveStream(ctx context.Context, reader io.Reader, outputArtifact *wfv1.Artifact) error {
	var req *http.Request
	var url string
	var err error
	if outputArtifact.Artifactory != nil && outputArtifact.HTTP == nil {
		url = outputArtifact.Artifactory.URL
		req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, reader)
		if err != nil {
			return err
		}
		req.SetBasicAuth(h.Username, h.Password)
	} else {
		url = outputArtifact.HTTP.URL
		req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, reader)
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

	res, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("saving stream to %s failed with reason: %s", url, res.Status)
	}
	return nil
}

// Delete is unsupported for the http artifacts
func (h *ArtifactDriver) Delete(ctx context.Context, s *wfv1.Artifact) error {
	return common.ErrDeleteNotSupported
}

func (h *ArtifactDriver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}

func (h *ArtifactDriver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	return false, errors.New(errors.CodeNotImplemented, "IsDirectory currently unimplemented for http")
}
