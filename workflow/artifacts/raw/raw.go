package raw

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

type ArtifactDriver struct{}

var _ common.ArtifactDriver = &ArtifactDriver{}

// Load Store raw content as artifact
func (a *ArtifactDriver) Load(ctx context.Context, artifact *wfv1.Artifact, path string) error {
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

func (a *ArtifactDriver) OpenStream(ctx context.Context, art *wfv1.Artifact) (io.ReadCloser, error) {
	// todo: this is a temporary implementation which loads file to disk first
	return common.LoadToStream(ctx, art, a)
}

// Save is unsupported for raw output artifacts
func (a *ArtifactDriver) Save(ctx context.Context, path string, artifact *wfv1.Artifact) error {
	return errors.Errorf(errors.CodeBadRequest, "Raw output artifacts unsupported")
}

// Delete is unsupported for raw output artifacts
func (a *ArtifactDriver) Delete(ctx context.Context, s *wfv1.Artifact) error {
	return common.ErrDeleteNotSupported
}

func (a *ArtifactDriver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}

func (a *ArtifactDriver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	return false, errors.New(errors.CodeNotImplemented, "IsDirectory currently unimplemented for raw")
}
