package logging

import (
	"context"
	"io"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// driver adds a logging interceptor to help diagnose issues with artifacts
type driver struct {
	common.ArtifactDriver
}

func New(d common.ArtifactDriver) common.ArtifactDriver {
	return &driver{d}
}

func (d *driver) Load(ctx context.Context, inputArtifact *wfv1.Artifact, path string) error {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "Loading artifact from %s", d.ArtifactDriver)
	t := time.Now()
	key, _ := inputArtifact.GetKey()
	err := d.ArtifactDriver.Load(ctx, inputArtifact, path)
	log.WithField("artifactName", inputArtifact.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info(ctx, "Load artifact")
	return err
}

func (d *driver) OpenStream(ctx context.Context, inputArtifact *wfv1.Artifact) (io.ReadCloser, error) {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "Opening stream from %s", d.ArtifactDriver)
	t := time.Now()
	key, _ := inputArtifact.GetKey()
	rc, err := d.ArtifactDriver.OpenStream(ctx, inputArtifact)
	log.WithField("artifactName", inputArtifact.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info(ctx, "Stream artifact")
	return rc, err
}

func (d *driver) Save(ctx context.Context, path string, outputArtifact *wfv1.Artifact) error {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "Saving artifact to %s", d.ArtifactDriver)
	t := time.Now()
	key, _ := outputArtifact.GetKey()
	err := d.ArtifactDriver.Save(ctx, path, outputArtifact)
	log.WithField("artifactName", outputArtifact.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info(ctx, "Save artifact")
	return err
}

func (d *driver) Delete(ctx context.Context, s *wfv1.Artifact) error {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "Deleting artifact from %s", d.ArtifactDriver)
	return d.ArtifactDriver.Delete(ctx, s)
}

func (d *driver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "Listing objects from %s", d.ArtifactDriver)
	t := time.Now()
	key, _ := artifact.GetKey()
	list, err := d.ArtifactDriver.ListObjects(ctx, artifact)
	log.WithField("artifactName", artifact.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info(ctx, "List objects")
	return list, err
}

func (d *driver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	log := logging.GetLoggerFromContext(ctx)
	log.Infof(ctx, "Checking if directory from %s", d.ArtifactDriver)
	t := time.Now()
	key, _ := artifact.GetKey()
	isDir, err := d.ArtifactDriver.IsDirectory(ctx, artifact)
	log.WithField("artifactName", artifact.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info(ctx, "Check if directory")
	return isDir, err
}
