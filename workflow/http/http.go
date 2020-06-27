package http

import (
	"io"
	"net/http"
	"os"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// HTTPArtifactDriver is the artifact driver for a HTTP URL
type HTTP struct {
	Username string
	Password string
}

// Load download artifacts from an HTTP URL
func (h *HTTP) Load(artifact *wfv1.HTTPArtifact, path string) error {
	lf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()
	req, err := http.NewRequest(http.MethodGet, artifact.URL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(h.Username, h.Password)
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
		return errors.InternalErrorf("HTTP request failed with reason:%s", res.Status)
	}
	_, err = io.Copy(lf, res.Body)
	return err
}

func (h *HTTP) Save(path string, artifact *wfv1.HTTPArtifact) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(artifact.Method, artifact.URL, f)
	if err != nil {
		return err
	}
	req.SetBasicAuth(h.Username, h.Password)
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("HTTP request %s failed with reason:%s", path, res.Status)
	}
	return nil
}
