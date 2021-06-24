package artifactory

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

type ArtifactDriver struct {
	Username string
	Password string
}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Download artifact from an artifactory URL
func (a *ArtifactDriver) Load(artifact *wfv1.Artifact, path string) error {
	lf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()

	req, err := http.NewRequest(http.MethodGet, artifact.Artifactory.URL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.Username, a.Password)
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
		return errors.InternalErrorf("loading file from artifactory failed with reason:%s", res.Status)
	}

	_, err = io.Copy(lf, res.Body)

	return err
}

// UpLoad artifact to an artifactory URL
func (a *ArtifactDriver) Save(path string, artifact *wfv1.Artifact) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, artifact.Artifactory.URL, f)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.Username, a.Password)
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("saving file %s to artifactory failed with reason:%s", path, res.Status)
	}
	return nil
}

func (a *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
