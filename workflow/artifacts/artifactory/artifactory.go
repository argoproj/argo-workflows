package artifactory

import (
	"io"
	"net/http"
	"os"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type ArtifactoryArtifactDriver struct {
	Username string
	Password string
}

// Download artifact from an artifactory URL
func (a *ArtifactoryArtifactDriver) Load(artifact *wfv1.Artifact, path string) error {

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
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.InternalErrorf("loading file from artifactory failed with reason:%s", res.Status)
	}

	_, err = io.Copy(lf, res.Body)

	return err
}

// UpLoad artifact to an artifactory URL
func (a *ArtifactoryArtifactDriver) Save(path string, artifact *wfv1.Artifact) error {

	f, err := os.Open(path)
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
