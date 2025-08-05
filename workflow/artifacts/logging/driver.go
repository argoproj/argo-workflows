package logging

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// driver adds a logging interceptor to help diagnose issues with artifacts
type driver struct {
	common.ArtifactDriver
}

func New(d common.ArtifactDriver) common.ArtifactDriver {
	return &driver{d}
}

func (d driver) Load(a *wfv1.Artifact, path string) error {
	t := time.Now()
	key, _ := a.GetKey()
	err := d.ArtifactDriver.Load(a, path)
	log.WithField("artifactName", a.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info("Load artifact")
	return err
}

func (d driver) OpenStream(a *wfv1.Artifact) (io.ReadCloser, error) {
	t := time.Now()
	key, _ := a.GetKey()
	rc, err := d.ArtifactDriver.OpenStream(a)
	log.WithField("artifactName", a.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info("Stream artifact")
	return rc, err
}

func (d driver) Save(path string, a *wfv1.Artifact) error {
	t := time.Now()
	key, _ := a.GetKey()
	err := d.ArtifactDriver.Save(path, a)
	log.WithField("artifactName", a.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info("Save artifact")
	return err
}

func (d driver) ListObjects(a *wfv1.Artifact) ([]string, error) {
	t := time.Now()
	key, _ := a.GetKey()
	list, err := d.ArtifactDriver.ListObjects(a)
	log.WithField("artifactName", a.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info("List objects")
	return list, err
}

func (d driver) IsDirectory(a *wfv1.Artifact) (bool, error) {
	t := time.Now()
	key, _ := a.GetKey()
	isDir, err := d.ArtifactDriver.IsDirectory(a)
	log.WithField("artifactName", a.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info("Check if directory")
	return isDir, err
}
