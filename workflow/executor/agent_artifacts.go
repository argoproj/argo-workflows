package executor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	argoerrs "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts"
	artifactcommon "github.com/argoproj/argo-workflows/v4/workflow/artifacts/common"
)

// Plugin templates declare input and output artifacts as container templates
// do, with paths rooted at a directory of the agent's workspace that is the
// task's own. The agent loads the input artifacts into the directory before
// offering the task to the plugins, hands the plugins a template whose
// artifact paths point into it, and uploads the output artifacts the plugin
// left there once the task is done: the init and wait containers' jobs for a
// pod. The workspace is shared with the plugins, so the agent never follows a
// symlink out of a task's directory.

func hasArtifacts(tmpl *wfv1.Template) bool {
	return len(tmpl.Inputs.Artifacts) > 0 || len(tmpl.Outputs.Artifacts) > 0
}

func (ae *AgentExecutor) executePluginTemplateWithArtifacts(ctx context.Context, nodeID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	dir, pluginTmpl, err := ae.prepareTask(ctx, nodeID, &tmpl)
	if err != nil {
		ae.removeTask(ctx, nodeID, dir)
		result.Phase = wfv1.NodeError
		result.Message = fmt.Sprintf("failed to prepare artifacts: %s", err)
		return 0, nil
	}
	requeue, err := ae.callPlugins(ctx, *pluginTmpl, result)
	switch {
	case err != nil:
		// processTask fails the task
		ae.removeTask(ctx, nodeID, dir)
	case result.Phase.Fulfilled(nil):
		if result.Phase.Completed() {
			ae.saveOutputArtifacts(ctx, nodeID, dir, &tmpl, result)
		}
		ae.removeTask(ctx, nodeID, dir)
	}
	return requeue, err
}

// prepareTask returns the directory of a plugin task, setting it up on the
// task's first run, and the template to offer the plugins, whose artifact
// paths point into the directory.
func (ae *AgentExecutor) prepareTask(ctx context.Context, nodeID string, tmpl *wfv1.Template) (string, *wfv1.Template, error) {
	dir, err := securejoin.SecureJoin(ae.workspace, nodeID)
	if err != nil {
		return "", nil, err
	}
	if _, ok := ae.preparedTasks.Load(nodeID); !ok {
		if err = ae.setUpTask(ctx, dir, tmpl); err != nil {
			return dir, nil, err
		}
		ae.preparedTasks.Store(nodeID, true)
	}
	pluginTmpl := tmpl.DeepCopy()
	for _, arts := range []wfv1.Artifacts{pluginTmpl.Inputs.Artifacts, pluginTmpl.Outputs.Artifacts} {
		for i := range arts {
			if arts[i].Path, err = taskPath(dir, arts[i].Path); err != nil {
				return dir, nil, err
			}
		}
	}
	return dir, pluginTmpl, nil
}

// setUpTask makes a plugin task's directory, loads the input artifacts into it
// and makes the directories the output artifacts go in. Everything in it is
// open to the plugins, which need not run as the agent's user.
func (ae *AgentExecutor) setUpTask(ctx context.Context, dir string, tmpl *wfv1.Template) error {
	// an agent that restarted may have left the directory part made
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := mkdirOpen(dir); err != nil {
		return err
	}
	for _, art := range tmpl.Inputs.Artifacts {
		if err := ae.loadInputArtifact(ctx, dir, tmpl.ArchiveLocation, art); err != nil {
			return err
		}
	}
	for _, art := range tmpl.Outputs.Artifacts {
		artPath, err := taskPath(dir, art.Path)
		if err != nil {
			return err
		}
		if err = mkdirShared(dir, filepath.Dir(artPath)); err != nil {
			return err
		}
	}
	return nil
}

// loadInputArtifact loads an input artifact of a plugin task to its path in
// the task's directory.
func (ae *AgentExecutor) loadInputArtifact(ctx context.Context, dir string, archiveLocation *wfv1.ArtifactLocation, art wfv1.Artifact) error {
	logger := logging.RequireLoggerFromContext(ctx).WithField("name", art.Name)
	if !art.HasLocationOrKey() {
		if art.Optional {
			logger.Info(ctx, "Ignoring optional artifact which was not supplied")
			return nil
		}
		return argoerrs.Errorf(argoerrs.CodeNotFound, "required artifact '%s' not supplied", art.Name)
	}
	artPath, err := taskPath(dir, art.Path)
	if err != nil {
		return err
	}
	driverArt := art.DeepCopy()
	if err = driverArt.Relocate(archiveLocation); err != nil {
		return fmt.Errorf("failed to load artifact '%s': %w", art.Name, err)
	}
	driver, err := ae.artifactDriver(ctx, driverArt)
	if err != nil {
		return err
	}
	if err = mkdirShared(dir, filepath.Dir(artPath)); err != nil {
		return err
	}
	// a previous run of the task may have loaded it already
	if err = os.RemoveAll(artPath); err != nil {
		return err
	}
	logger.WithField("path", artPath).Info(ctx, "Downloading artifact")
	tempArtPath := artPath + ".tmp"
	if err = driver.Load(ctx, driverArt, tempArtPath); err != nil {
		if art.Optional && argoerrs.IsCode(argoerrs.CodeNotFound, err) {
			logger.Info(ctx, "Skipping optional input artifact that was not found")
			return nil
		}
		return fmt.Errorf("artifact %s failed to load: %w", art.Name, err)
	}
	isTar, isZip, err := artifactArchiveFormat(ctx, art, tempArtPath)
	if err != nil {
		return err
	}
	if err = unarchive(ctx, isTar, isZip, tempArtPath, artPath); err != nil {
		return err
	}
	if art.Mode != nil {
		return chmod(artPath, *art.Mode, art.RecurseMode)
	}
	return share(artPath)
}

// saveOutputArtifacts uploads the output artifacts a plugin task left in its
// directory and reports them in the task's result, together with any the
// plugin stored and reported itself. Only optional artifacts may be missing
// from a task that succeeded; one that failed has whatever it left uploaded.
func (ae *AgentExecutor) saveOutputArtifacts(ctx context.Context, nodeID, dir string, tmpl *wfv1.Template, result *wfv1.NodeResult) {
	if len(tmpl.Outputs.Artifacts) == 0 {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	outputsDir := filepath.Join(ae.outputsDir, nodeID)
	defer func() {
		if err := os.RemoveAll(outputsDir); err != nil {
			logger.WithError(err).Warn(ctx, "Failed to remove archived output artifacts")
		}
	}()

	var reported wfv1.Artifacts
	if result.Outputs != nil {
		reported = result.Outputs.Artifacts
	}
	var saved wfv1.Artifacts
	var errs []string
	for _, art := range tmpl.Outputs.Artifacts {
		if r := reported.GetArtifactByName(art.Name); r != nil && r.HasLocationOrKey() {
			saved = append(saved, *r)
			continue
		}
		mayBeMissing := art.Optional || result.Phase != wfv1.NodeSucceeded
		ok, err := ae.saveOutputArtifact(ctx, dir, outputsDir, tmpl.ArchiveLocation, &art, mayBeMissing)
		if err != nil {
			logger.WithField("name", art.Name).WithError(err).Error(ctx, "Failed to save output artifact")
			errs = append(errs, err.Error())
		} else if ok {
			saved = append(saved, art)
		}
	}
	for _, r := range reported {
		if tmpl.Outputs.Artifacts.GetArtifactByName(r.Name) == nil {
			saved = append(saved, r)
		}
	}

	if len(saved) > 0 || result.Outputs != nil {
		if result.Outputs == nil {
			result.Outputs = &wfv1.Outputs{}
		}
		result.Outputs.Artifacts = saved
	}
	if len(errs) > 0 && result.Phase == wfv1.NodeSucceeded {
		result.Phase = wfv1.NodeError
		result.Message = fmt.Sprintf("failed to save output artifacts: %s", strings.Join(errs, "; "))
	}
}

// saveOutputArtifact uploads an output artifact from its path in the task's
// directory, archived in outputsDir as its archive strategy asks, keying it
// under the archive location unless it has a key of its own. It reports
// whether it uploaded the artifact: one that may be missing and is, is not.
func (ae *AgentExecutor) saveOutputArtifact(ctx context.Context, dir, outputsDir string, archiveLocation *wfv1.ArtifactLocation, art *wfv1.Artifact, mayBeMissing bool) (bool, error) {
	logger := logging.RequireLoggerFromContext(ctx).WithField("name", art.Name)
	srcPath, err := taskPath(dir, art.Path)
	if err != nil {
		return false, err
	}
	if art.GetArchive().None != nil {
		// the driver uploads the files themselves, and would follow a symlink
		// out of the task's directory
		if err = rejectSymlinks(srcPath); err != nil {
			return false, err
		}
	}
	if err = os.MkdirAll(outputsDir, 0o700); err != nil {
		return false, err
	}
	fileName, localArtPath, err := archiveLocalPath(ctx, art, srcPath, outputsDir)
	switch {
	case argoerrs.IsCode(argoerrs.CodeNotFound, err) && mayBeMissing:
		logger.WithError(err).Warn(ctx, "Ignoring output artifact which does not exist in path")
		return false, nil
	case argoerrs.IsCode(argoerrs.CodeNotFound, err):
		return false, argoerrs.Errorf(argoerrs.CodeNotFound, "required artifact '%s' not found at %s", art.Name, art.Path)
	case err != nil:
		return false, err
	}
	if err = keyArtifact(art, archiveLocation, fileName); err != nil {
		return false, err
	}
	driverArt := art.DeepCopy()
	if err = driverArt.Relocate(archiveLocation); err != nil {
		return false, err
	}
	driver, err := ae.artifactDriver(ctx, driverArt)
	if err != nil {
		return false, err
	}
	if err = driver.Save(ctx, localArtPath, driverArt); err != nil {
		return false, fmt.Errorf("failed to save artifact '%s': %w", art.Name, err)
	}
	logger.Info(ctx, "Successfully saved artifact")
	return true, nil
}

// removeTask forgets a plugin task that is done and removes its directory.
// Whatever a plugin running as another user made in it may not be removable;
// that goes with the agent pod.
func (ae *AgentExecutor) removeTask(ctx context.Context, nodeID, dir string) {
	ae.preparedTasks.Delete(nodeID)
	if dir == "" {
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Failed to remove the task's directory")
	}
}

func (ae *AgentExecutor) artifactDriver(ctx context.Context, art *wfv1.Artifact) (artifactcommon.ArtifactDriver, error) {
	if art.Plugin != nil {
		// artifact plugins run as sidecars of the pods that use them, and the
		// agent pod has none
		return nil, argoerrs.Errorf(argoerrs.CodeBadRequest, "artifact '%s' is stored by artifact plugin %s, which plugin templates cannot use", art.Name, art.Plugin.Name)
	}
	driver, err := ae.newDriver(ctx, art, ae.resources)
	if errors.Is(err, artifacts.ErrUnsupportedDriver) {
		return nil, argoerrs.Errorf(argoerrs.CodeBadRequest, "Unsupported artifact driver for %s", art.Name)
	}
	return driver, err
}

// taskPath returns where an artifact path of a plugin task is, the task's
// directory standing for the root of the plugin's file system. Symlinks
// resolve within the directory too, so no path leads out of it.
func taskPath(dir, artPath string) (string, error) {
	return securejoin.SecureJoin(dir, artPath)
}

// mkdirShared makes the directory dir and the directory p within it, with
// the parents p lacks, open to every user. A p outside dir is left be.
func mkdirShared(dir, p string) error {
	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return err
	}
	if !filepath.IsLocal(rel) {
		return nil
	}
	if err := mkdirOpen(dir); err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	for name := range strings.SplitSeq(rel, string(filepath.Separator)) {
		dir = filepath.Join(dir, name)
		if err := mkdirOpen(dir); err != nil {
			return err
		}
	}
	return nil
}

// mkdirOpen makes the directory p unless it exists, and opens it to every user.
func mkdirOpen(p string) error {
	if err := os.Mkdir(p, 0o777); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	info, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", p)
	}
	// Mkdir is subject to the umask
	return os.Chmod(p, 0o777)
}

// share opens up the files and directories at and under p to every user,
// leaving symlinks be.
func share(p string) error {
	return filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.Type()&fs.ModeSymlink != 0:
			return nil
		case d.IsDir():
			return os.Chmod(path, 0o777)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode().Perm()|0o666)
	})
}

// rejectSymlinks fails for a symlink under p, which an artifact uploaded with
// no archive cannot hold.
func rejectSymlinks(p string) error {
	err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return argoerrs.Errorf(argoerrs.CodeBadRequest, "%s is a symlink, which an artifact with archive none cannot hold", path)
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		// archiving reports the artifact missing
		return nil
	}
	return err
}

// agentResources gives artifact drivers secrets from where the controller
// mounted them in the agent pod, which are the workflow's artifact
// repository's, and any other secret, and configmaps, from the Kubernetes API.
type agentResources struct {
	clientSet kubernetes.Interface
	namespace string
	secretDir string
}

func (r agentResources) GetSecret(ctx context.Context, name, key string) (string, error) {
	if !isPathElement(name) || !isPathElement(key) {
		return "", argoerrs.Errorf(argoerrs.CodeBadRequest, "invalid secret name %q or key %q", name, key)
	}
	data, err := os.ReadFile(filepath.Join(r.secretDir, name, key))
	if err == nil {
		return string(data), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	secret, err := util.GetSecrets(ctx, r.clientSet, r.namespace, name, key)
	if err != nil {
		return "", fmt.Errorf("secret %q is not mounted in the agent pod, and reading it from the Kubernetes API failed: %w", name, err)
	}
	return string(secret), nil
}

func (r agentResources) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	configMap, err := r.clientSet.CoreV1().ConfigMaps(r.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := configMap.Data[key]
	if !ok {
		return "", argoerrs.Errorf(argoerrs.CodeBadRequest, "configmap '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// isPathElement reports whether s names a file in a directory, and nothing
// outside it.
func isPathElement(s string) bool {
	return filepath.IsLocal(s) && filepath.Base(s) == s
}
