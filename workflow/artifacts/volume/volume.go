package volume

import (
	"path/filepath"
	"strings"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
)

var availableCompressions = []string{"tar.gz"}

// ArtifactDriver is a driver for volume
type ArtifactDriver struct {
	MountPath string
}

// ValidateArtifact validates volume artifact
func ValidateArtifact(errPrefix string, art *wfv1.VolumeArtifact) error {
	if art.Name == "" {
		return errors.Errorf(errors.CodeBadRequest, "%s.name is required", errPrefix)
	}
	if strings.Contains(art.SubPath, "..") {
		return errors.Errorf(errors.CodeBadRequest, "%s.subPath is invalid", errPrefix)
	}
	if art.Path == "" {
		return errors.Errorf(errors.CodeBadRequest, "%s.path is required", errPrefix)
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
	log.Debugf("Loading file %s to %s", srcpath, path)
	err := util.CopyFile(path, srcpath)
	return err
}

// Save copies a file to a volume
func (driver *ArtifactDriver) Save(path string, artifact *wfv1.Artifact) error {
	dstpath := filepath.Join(driver.MountPath, artifact.Volume.Name, artifact.Volume.SubPath, artifact.Volume.Path)
	log.Debugf("Saving file %s to %s", path, dstpath)
	err := util.CopyFile(dstpath, path)
	return err
}
