package raw

import (
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"os"
)

type RawArtifactDriver struct {
}

// Store raw content as artifact
func (a *RawArtifactDriver) Load(artifact *wfv1.Artifact, path string) error {
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
func (g *RawArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Raw output artifacts unsupported")
}
