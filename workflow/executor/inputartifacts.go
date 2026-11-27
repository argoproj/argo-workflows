package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// StageInputArtifacts links each input artifact from paths.InputArtifacts
// into place and re-enters the working directory afterwards, stepping off it
// for the duration: linking replaces the cwd when an artifact's path is the
// container's workingDir, Windows refuses to delete a directory in use as a
// working directory, and a child forked with the deleted directory as cwd
// would see getcwd() fail and relative paths resolve to nothing. The final
// chdir follows the symlink to whatever now sits at the path. If linking
// fails, the original working directory is restored before returning.
//
// The chdir is process-wide, so StageInputArtifacts is for a process that
// runs one task (the emissary). An in-process caller should use
// linkInputArtifacts and give the command its working directory instead.
func StageInputArtifacts(ctx context.Context, paths TaskPaths, tmpl *wfv1.Template) error {
	origWd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to read working directory before staging input artifacts: %w", err)
	}
	if err := os.Chdir(paths.VarRunArgo); err != nil {
		return fmt.Errorf("failed to leave working directory before staging input artifacts: %w", err)
	}
	if err := linkInputArtifacts(ctx, paths.InputArtifacts, tmpl); err != nil {
		_ = os.Chdir(origWd)
		return err
	}
	if err := os.Chdir(origWd); err != nil {
		return fmt.Errorf("failed to re-enter working directory %q after staging input artifacts (an input artifact staged at the workingDir path must be a directory): %w", origWd, err)
	}
	return nil
}

// linkInputArtifacts creates a symlink at each input artifact's path
// pointing to the file Prepare loaded at inputArtifactsDir/<name>
// (/argo/inputs/artifacts in a pod). In a pod this replaces the legacy SubPath bind-mount-per-artifact scheme,
// which can't be used in init-less mode because kubelet pre-creates SubPath
// entries as empty directories before supervisor can write the real file.
//
// Behavior notes for workflow authors: in init-less mode art.Path is a
// symlink rather than a regular file. `cat`, `open()`, `tar`, `cp`,
// redirection, etc. all follow symlinks transparently and see identical
// content. Code that calls `lstat`/`readlink` on art.Path will observe a
// symlink rather than a regular file. `rm art.Path` removes the symlink
// only; the underlying artifact stays in the shared emptyDir.
//
// Overlapping user volumes are handled by the executor on the write side
// (Prepare writes under TaskPaths.MainFS instead of inputArtifactsDir),
// so no entry appears in the input-artifacts directory
// and we skip the symlink — main already sees the file at art.Path via
// its user volume.
func linkInputArtifacts(ctx context.Context, inputArtifactsDir string, tmpl *wfv1.Template) error {
	logger := logging.RequireLoggerFromContext(ctx)
	for _, art := range tmpl.Inputs.Artifacts {
		src := filepath.Join(inputArtifactsDir, art.Name)
		if _, statErr := os.Lstat(src); statErr != nil {
			if os.IsNotExist(statErr) {
				logger.WithFields(logging.Fields{"name": art.Name, "path": art.Path}).Info(ctx, "no input-artifacts entry (optional or overlap) — skipping symlink")
				continue
			}
			return fmt.Errorf("failed to stat input artifact %q at %s: %w", art.Name, src, statErr)
		}
		dst := art.Path
		if dst == "" {
			continue
		}
		if parent := filepath.Dir(dst); parent != "" && parent != "/" {
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory for artifact %q at %s: %w", art.Name, dst, err)
			}
		}
		// If nothing exists at art.Path, just create the symlink. Creating is
		// always safe — os.Symlink returns EEXIST rather than overwriting and the
		// MkdirAll above only ever creates — so even when art.Path resolves into a
		// user volume we deliberately let the artifact land there (the user asked
		// for it). Only an *overwrite* can destroy data, and that is gated below.
		if _, err := os.Lstat(dst); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat artifact path %q at %s: %w", art.Name, dst, err)
			}
		} else {
			// Something is already at art.Path. Replacing it (os.RemoveAll then
			// symlink) reproduces the legacy SubPath mount's shadowing — but only
			// when it is safe. RemoveAll resolves symlinks in the parent chain, so
			// resolve the parent to find where the delete would actually land
			// (resolve the parent, not the final element, so an image symlink *at*
			// art.Path is just unlinked rather than followed). If that resolved
			// path overlaps a user-declared volume, clearing it would recurse into
			// and destroy a live PVC/hostPath/emptyDir, so refuse. Benign system
			// mounts (tmpfs /run, the overlay rootfs) are not declared user volumes
			// and so remain safe to shadow.
			realParent, evalErr := filepath.EvalSymlinks(filepath.Dir(dst))
			if evalErr != nil {
				return fmt.Errorf("failed to resolve parent of artifact path %q at %s: %w", art.Name, dst, evalErr)
			}
			resolved := filepath.Join(realParent, filepath.Base(dst))
			if mnt := common.FindOverlappingVolume(tmpl, resolved); mnt != nil {
				return fmt.Errorf("refusing to stage input artifact %q at %s: it resolves to %s inside volume mount %q (%s), and clearing it would destroy the mounted volume; change the artifact path or volume mount so they do not overlap", art.Name, dst, resolved, mnt.Name, mnt.MountPath)
			}
			if mnt := common.FindVolumeMountNestedUnderPath(tmpl, resolved); mnt != nil {
				return fmt.Errorf("refusing to stage input artifact %q at %s: it resolves to %s which contains volume mount %q (%s), and clearing it would destroy the mounted volume; change the artifact path or volume mount so they do not overlap", art.Name, dst, resolved, mnt.Name, mnt.MountPath)
			}
			if rmErr := os.RemoveAll(dst); rmErr != nil {
				return fmt.Errorf("failed to clear existing path for artifact %q at %s: %w", art.Name, dst, rmErr)
			}
		}
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("failed to symlink input artifact %q (%s -> %s): %w", art.Name, dst, src, err)
		}
		logger.WithFields(logging.Fields{"name": art.Name, "src": src, "dst": dst}).Debug(ctx, "linked input artifact")
	}
	return nil
}
