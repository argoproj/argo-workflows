package volume

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/common"
)

var availableCompressions = []string{"tar.gz"}

// ArtifactDriver is a driver for volume
type ArtifactDriver struct {
	MountPath string
}

// ValidateArtifact validates volume artifact
func ValidateArtifact(errPrefix string, art *wfv1.VolumeArtifact) error {
	if strings.Contains(art.SubPath, "..") {
		return errors.Errorf(errors.CodeBadRequest, "%s.subPath is invalid", errPrefix)
	}
	if strings.Contains(art.Path, "..") {
		return errors.Errorf(errors.CodeBadRequest, "%s.path is invalid", errPrefix)
	}
	return nil
}

// CreateDriver constructs ArtifactDriver
func CreateDriver(ci common.ResourceInterface, art *wfv1.VolumeArtifact) (*ArtifactDriver, error) {
	driver := ArtifactDriver{
		MountPath: common.VolumeArtifactMountPath,
	}
	return &driver, nil
}

// Load copies a file from a volume
func (driver *ArtifactDriver) Load(artifact *wfv1.Artifact, path string) error {
	srcpath := filepath.Join(driver.MountPath, artifact.Volume.Name, artifact.Volume.SubPath, artifact.Volume.Path)

	srcf, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer util.Close(srcf)

	dstf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer util.Close(dstf)

	_, err = io.Copy(dstf, srcf)

	return err
}

// Save copies a file to a volume
func (driver *ArtifactDriver) Save(path string, artifact *wfv1.Artifact) error {
	srcf, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = srcf.Close()
	}()

	dstpath := filepath.Join(driver.MountPath, artifact.Volume.Name, artifact.Volume.SubPath, artifact.Volume.Path)
	dstf, err := os.Create(dstpath)
	if err != nil {
		return err
	}
	defer func() {
		_ = dstf.Close()
	}()

	_, err = io.Copy(dstf, srcf)

	return err
}
