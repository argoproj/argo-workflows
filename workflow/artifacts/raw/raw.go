package raw

import (
	"fmt"
	"os"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

type ArtifactDriver struct{}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Store raw content as artifact
func (a *ArtifactDriver) Load(artifact *wfv1.Artifact, path string) error {
	lf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()

	_, err = lf.WriteString(artifact.Raw.Data)
	return err
}

// Save is unsupported for raw output artifacts
func (g *ArtifactDriver) Save(string, *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Raw output artifacts unsupported")
}

func (a *ArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}
