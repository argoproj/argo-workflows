package k8s

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/archive"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

// PodSink stages a task's outputs under VarRunArgo/outputs, where the wait or
// supervisor container collects them. Only the main container's outputs are
// staged; logs are already in the task's WorkDir, so they need nothing.
//
// theory-debt: the sink is built per task, holding that task's template (for
// the overlapping-volume check) rather than looking it up by nodeID.
type PodSink struct {
	VarRunArgo    string
	ContainerName string
	Template      *wfv1.Template
}

var _ maindriver.ResultSink = PodSink{}

func (s PodSink) Put(ctx context.Context, _ string, out maindriver.Output) error {
	if s.ContainerName != common.MainContainerName {
		return nil
	}
	switch out.Kind {
	case maindriver.OutputParameter:
		return s.saveParameter(ctx, out.Path)
	case maindriver.OutputArtifact:
		return s.saveArtifact(ctx, out.Path)
	default:
		return nil
	}
}

func (s PodSink) saveArtifact(ctx context.Context, srcPath string) error {
	logger := logging.RequireLoggerFromContext(ctx)

	if common.FindOverlappingVolume(s.Template, srcPath) != nil {
		logger.WithField("srcPath", srcPath).Info(ctx, "no need to save artifact - on overlapping volume")
		return nil
	}
	if _, err := os.Stat(srcPath); os.IsNotExist(err) { // might be optional, so we ignore
		logger.WithField("srcPath", srcPath).WithError(err).Warn(ctx, "cannot save artifact")
		return nil
	}
	dstPath := filepath.Join(s.VarRunArgo, "/outputs/artifacts/", strings.TrimSuffix(srcPath, "/")+".tgz")
	logger.WithFields(logging.Fields{
		"src": srcPath,
		"dst": dstPath,
	}).Info(ctx, "saving artifact")
	z := filepath.Dir(dstPath)
	if err := os.MkdirAll(z, 0o755); err != nil { // chmod rwxr-xr-x
		return fmt.Errorf("failed to create directory %s: %w", z, err)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination %s: %w", dstPath, err)
	}
	defer func() { _ = dst.Close() }()
	if err = archive.TarGzToWriter(ctx, srcPath, gzip.DefaultCompression, dst); err != nil {
		return fmt.Errorf("failed to tarball the output %s to %s: %w", srcPath, dstPath, err)
	}
	if err = dst.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", dstPath, err)
	}
	return nil
}

func (s PodSink) saveParameter(ctx context.Context, srcPath string) error {
	logger := logging.RequireLoggerFromContext(ctx)

	if common.FindOverlappingVolume(s.Template, srcPath) != nil {
		logger.WithField("src", srcPath).Info(ctx, "no need to save parameter - on overlapping volume")
		return nil
	}
	src, err := os.Open(filepath.Clean(srcPath))
	if os.IsNotExist(err) { // might be optional, so we ignore
		logger.WithField("src", srcPath).WithError(err).Warn(ctx, "cannot save parameter, does not exist")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()
	dstPath := s.VarRunArgo + "/outputs/parameters/" + srcPath
	logger.WithFields(logging.Fields{
		"src": srcPath,
		"dst": dstPath,
	}).Info(ctx, "saving parameter")
	z := filepath.Dir(dstPath)
	if mkdirErr := os.MkdirAll(z, 0o755); mkdirErr != nil { // chmod rwxr-xr-x
		return fmt.Errorf("failed to create directory %s: %w", z, mkdirErr)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", srcPath, err)
	}
	defer func() { _ = dst.Close() }()
	if _, err = io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
	}
	if err = dst.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", dstPath, err)
	}
	return nil
}
