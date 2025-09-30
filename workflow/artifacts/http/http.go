package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	workflowcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
)

// ArtifactDriver is the artifact driver for artifactory and http URLs
type ArtifactDriver struct {
	Username  string
	Password  string
	Client    *http.Client
	ClientSet kubernetes.Interface
	Namespace string
}

var _ common.ArtifactDriver = &ArtifactDriver{}

func (h *ArtifactDriver) retrieveContent(ctx context.Context, inputArtifact *wfv1.Artifact) (http.Response, error) {
	var req *http.Request
	var url string
	var err error
	var auth *wfv1.HTTPAuth

	if inputArtifact.Artifactory != nil && inputArtifact.HTTP == nil {
		url = inputArtifact.Artifactory.URL
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return http.Response{}, err
		}
		// For backward compatibility, use Username/Password for Artifactory
		if h.Username != "" && h.Password != "" {
			req.SetBasicAuth(h.Username, h.Password)
		}
	} else if inputArtifact.Artifactory == nil && inputArtifact.HTTP != nil {
		url = inputArtifact.HTTP.URL
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return http.Response{}, err
		}

		// Add headers
		for _, header := range inputArtifact.HTTP.Headers {
			req.Header.Add(header.Name, header.Value)
		}

		// Use new unified auth if available
		if inputArtifact.HTTP.Auth != nil {
			auth = inputArtifact.HTTP.Auth
		} else if h.Username != "" && h.Password != "" {
			// Backward compatibility: use driver-level username/password
			req.SetBasicAuth(h.Username, h.Password)
		}
	} else {
		return http.Response{}, errors.InternalErrorf("Either Artifactory or HTTP artifact needs to be configured")
	}

	// Apply unified authentication if configured
	if auth != nil && h.ClientSet != nil {
		if err := workflowcommon.ApplyHTTPAuth(ctx, req, auth, h.ClientSet, h.Namespace); err != nil {
			return http.Response{}, fmt.Errorf("failed to apply HTTP authentication: %w", err)
		}
	}

	// Use appropriate HTTP client
	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}

	// Create client with auth if needed
	if auth != nil && h.ClientSet != nil {
		client, err = workflowcommon.CreateHTTPClientWithAuth(ctx, auth, false, h.ClientSet, h.Namespace)
		if err != nil {
			return http.Response{}, fmt.Errorf("failed to create authenticated HTTP client: %w", err)
		}
	}

	// Note that we will close the response body in either `Load()`
	// or `ArtifactServer.returnArtifact()`, which is the caller of `OpenStream()`.
	res, err := client.Do(req) //nolint:bodyclose
	if err != nil {
		return http.Response{}, err
	}
	if res.StatusCode == 404 {
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
	var auth *wfv1.HTTPAuth

	if outputArtifact.Artifactory != nil && outputArtifact.HTTP == nil {
		url = outputArtifact.Artifactory.URL
		req, err = http.NewRequest(http.MethodPut, url, f)
		if err != nil {
			return err
		}
		// For backward compatibility, use Username/Password for Artifactory
		if h.Username != "" && h.Password != "" {
			req.SetBasicAuth(h.Username, h.Password)
		}
	} else {
		url = outputArtifact.HTTP.URL
		req, err = http.NewRequest(http.MethodPut, url, f)
		if err != nil {
			return err
		}

		// Add headers
		for _, header := range outputArtifact.HTTP.Headers {
			req.Header.Add(header.Name, header.Value)
		}

		// Use new unified auth if available
		if outputArtifact.HTTP.Auth != nil {
			auth = outputArtifact.HTTP.Auth
		} else if h.Username != "" && h.Password != "" {
			// Backward compatibility: use driver-level username/password
			req.SetBasicAuth(h.Username, h.Password)
		}
	}

	// Apply unified authentication if configured
	if auth != nil && h.ClientSet != nil {
		if err := workflowcommon.ApplyHTTPAuth(ctx, req, auth, h.ClientSet, h.Namespace); err != nil {
			return fmt.Errorf("failed to apply HTTP authentication: %w", err)
		}
	}

	// we set the GetBody func of the request in order to enable following 307 POST/PUT redirects, needed e.g. for webHDFS
	req.GetBody = func() (io.ReadCloser, error) {
		return os.Open(cleanPath)
	}

	// Use appropriate HTTP client
	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}

	// Create client with auth if needed
	if auth != nil && h.ClientSet != nil {
		client, err = workflowcommon.CreateHTTPClientWithAuth(ctx, auth, false, h.ClientSet, h.Namespace)
		if err != nil {
			return fmt.Errorf("failed to create authenticated HTTP client: %w", err)
		}
	}

	res, err := client.Do(req)
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
