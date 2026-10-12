package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
// pod.
//
// Any plugin can change anything in the workspace at any time, so the agent
// only touches it through an os.Root, which neither a symlink nor ".." leads
// out of at any step. Artifact drivers and archives work with paths instead,
// so the agent downloads, archives and uploads artifacts in a staging
// directory of its own, out of the plugins' reach, copying them between it
// and the workspace.

// agentStagingDir is the agent's staging directory, in its /tmp volume.
const agentStagingDir = "/tmp/argo/agent"

func hasArtifacts(tmpl *wfv1.Template) bool {
	return len(tmpl.Inputs.Artifacts) > 0 || len(tmpl.Outputs.Artifacts) > 0
}

func (ae *AgentExecutor) executePluginTemplateWithArtifacts(ctx context.Context, nodeID string, tmpl wfv1.Template, result *wfv1.NodeResult) (time.Duration, error) {
	// the node ID names the task's directories
	if !isPathElement(nodeID) {
		result.Phase = wfv1.NodeError
		result.Message = fmt.Sprintf("invalid node ID %q", nodeID)
		return 0, nil
	}
	pluginTmpl, err := ae.prepareTask(ctx, nodeID, &tmpl)
	if err != nil {
		ae.removeTask(ctx, nodeID)
		result.Phase = wfv1.NodeError
		result.Message = fmt.Sprintf("failed to prepare artifacts: %s", err)
		return 0, nil
	}
	requeue, err := ae.callPlugins(ctx, *pluginTmpl, result)
	switch {
	case err != nil:
		// processTask fails the task
		ae.removeTask(ctx, nodeID)
	case result.Phase.Fulfilled(nil):
		if result.Phase.Completed() {
			ae.saveOutputArtifacts(ctx, nodeID, &tmpl, result)
		}
		ae.removeTask(ctx, nodeID)
	}
	return requeue, err
}

// prepareTask sets a plugin task's directory up on the task's first run, and
// returns the template to offer the plugins, whose artifact paths point into
// the directory. A task that is set up is marked so in its staging directory,
// which a restarted agent finds as it finds the workspace: both outlive the
// agent's container.
func (ae *AgentExecutor) prepareTask(ctx context.Context, nodeID string, tmpl *wfv1.Template) (*wfv1.Template, error) {
	marker := filepath.Join(ae.stagingDir, nodeID, "prepared")
	_, err := os.Stat(marker)
	if errors.Is(err, fs.ErrNotExist) {
		if err = ae.setUpTask(ctx, nodeID, tmpl); err != nil {
			return nil, err
		}
		err = os.WriteFile(marker, nil, 0o600)
	}
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(ae.workspace, nodeID)
	pluginTmpl := tmpl.DeepCopy()
	for _, arts := range []wfv1.Artifacts{pluginTmpl.Inputs.Artifacts, pluginTmpl.Outputs.Artifacts} {
		for i := range arts {
			arts[i].Path = filepath.Join(dir, taskPath(arts[i].Path))
		}
	}
	return pluginTmpl, nil
}

// setUpTask makes a plugin task's directory, loads the input artifacts into it
// and makes the directories the output artifacts go in. Everything in it is
// open to the plugins, which need not run as the agent's user.
func (ae *AgentExecutor) setUpTask(ctx context.Context, nodeID string, tmpl *wfv1.Template) error {
	staging := filepath.Join(ae.stagingDir, nodeID)
	// an agent that restarted may have left the task part set up
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return err
	}
	workspace, err := os.OpenRoot(ae.workspace)
	if err != nil {
		return err
	}
	defer workspace.Close()
	if err = workspace.RemoveAll(nodeID); err != nil {
		return err
	}
	if err = mkdirShared(workspace, nodeID); err != nil {
		return err
	}
	dir, err := workspace.OpenRoot(nodeID)
	if err != nil {
		return err
	}
	defer dir.Close()
	for i, art := range tmpl.Inputs.Artifacts {
		if err = ae.loadInputArtifact(ctx, dir, filepath.Join(staging, "inputs", strconv.Itoa(i)), tmpl.ArchiveLocation, art); err != nil {
			return err
		}
	}
	for _, art := range tmpl.Outputs.Artifacts {
		if err = mkdirAllShared(dir, filepath.Dir(taskPath(art.Path))); err != nil {
			return err
		}
	}
	return nil
}

// loadInputArtifact loads an input artifact of a plugin task in staging, and
// copies it to its path in the task's directory, dir.
func (ae *AgentExecutor) loadInputArtifact(ctx context.Context, dir *os.Root, staging string, archiveLocation *wfv1.ArtifactLocation, art wfv1.Artifact) error {
	logger := logging.RequireLoggerFromContext(ctx).WithField("name", art.Name)
	if !art.HasLocationOrKey() {
		if art.Optional {
			logger.Info(ctx, "Ignoring optional artifact which was not supplied")
			return nil
		}
		return argoerrs.Errorf(argoerrs.CodeNotFound, "required artifact '%s' not supplied", art.Name)
	}
	name := taskPath(art.Path)
	if name == "." {
		return argoerrs.Errorf(argoerrs.CodeBadRequest, "input artifact '%s' cannot be loaded at %s, which is the whole task's directory", art.Name, art.Path)
	}
	driverArt := art.DeepCopy()
	if err := driverArt.Relocate(archiveLocation); err != nil {
		return fmt.Errorf("failed to load artifact '%s': %w", art.Name, err)
	}
	driver, err := ae.artifactDriver(ctx, driverArt)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(staging, 0o700); err != nil {
		return err
	}
	defer func() {
		if removeErr := os.RemoveAll(staging); removeErr != nil {
			logger.WithError(removeErr).Warn(ctx, "Failed to remove the downloaded artifact")
		}
	}()
	logger.WithField("path", art.Path).Info(ctx, "Downloading artifact")
	tempArtPath := filepath.Join(staging, "artifact.tmp")
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
	artPath := filepath.Join(staging, "artifact")
	if err = unarchive(ctx, isTar, isZip, tempArtPath, artPath); err != nil {
		return err
	}
	if err = mkdirAllShared(dir, filepath.Dir(name)); err != nil {
		return err
	}
	// a plugin may have put something there
	if err = dir.RemoveAll(name); err != nil {
		return err
	}
	return copyIn(artPath, dir, name, func(top bool, mode fs.FileMode) fs.FileMode {
		return inputMode(&art, top, mode)
	})
}

// inputMode returns the mode a file or directory of an input artifact, with
// the mode it was loaded with, gets in a plugin task's directory: the
// artifact's mode for the artifact itself or, if it recurses, for everything
// in it too; else one open to every user, as plugins need not run as the
// agent's user.
func inputMode(art *wfv1.Artifact, top bool, mode fs.FileMode) fs.FileMode {
	switch {
	case art.Mode != nil && (top || art.RecurseMode):
		return fs.FileMode(*art.Mode)
	case art.Mode != nil:
		return mode.Perm()
	case mode.IsDir():
		return 0o777
	default:
		return mode.Perm() | 0o666
	}
}

// saveOutputArtifacts uploads the output artifacts a plugin task left in its
// directory and reports them in the task's result, together with any the
// plugin stored and reported itself. Only optional artifacts may be missing
// from a task that succeeded; one that failed has whatever it left uploaded.
func (ae *AgentExecutor) saveOutputArtifacts(ctx context.Context, nodeID string, tmpl *wfv1.Template, result *wfv1.NodeResult) {
	if len(tmpl.Outputs.Artifacts) == 0 {
		return
	}
	logger := logging.RequireLoggerFromContext(ctx)
	staging := filepath.Join(ae.stagingDir, nodeID, "outputs")
	// an agent that restarted may have left some staged
	if err := os.RemoveAll(staging); err != nil {
		logger.WithError(err).Warn(ctx, "Failed to remove previously staged output artifacts")
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			logger.WithError(err).Warn(ctx, "Failed to remove staged output artifacts")
		}
	}()

	var reported wfv1.Artifacts
	if result.Outputs != nil {
		reported = result.Outputs.Artifacts
	}
	var saved wfv1.Artifacts
	var errs []string
	for i, art := range tmpl.Outputs.Artifacts {
		if r := reported.GetArtifactByName(art.Name); r != nil && r.HasLocationOrKey() {
			saved = append(saved, *r)
			continue
		}
		mayBeMissing := art.Optional || result.Phase != wfv1.NodeSucceeded
		ok, err := ae.saveOutputArtifact(ctx, nodeID, filepath.Join(staging, strconv.Itoa(i)), tmpl.ArchiveLocation, &art, mayBeMissing)
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

// saveOutputArtifact copies an output artifact from its path in a plugin
// task's directory to staging, archives it there as its archive strategy
// asks, and uploads it, keying it under the archive location unless it has a
// key of its own. It reports whether it uploaded the artifact: one that may be
// missing and is, is not.
func (ae *AgentExecutor) saveOutputArtifact(ctx context.Context, nodeID, staging string, archiveLocation *wfv1.ArtifactLocation, art *wfv1.Artifact, mayBeMissing bool) (bool, error) {
	logger := logging.RequireLoggerFromContext(ctx).WithField("name", art.Name)
	name := taskPath(art.Path)
	// named as it is in the task's directory, which an archive keeps
	copied := filepath.Join(staging, "files", filepath.Base(filepath.Join(nodeID, name)))
	if err := os.MkdirAll(filepath.Dir(copied), 0o700); err != nil {
		return false, err
	}
	err := ae.copyOutputArtifact(nodeID, name, copied, art.GetArchive().None != nil)
	switch {
	case errors.Is(err, fs.ErrNotExist) && mayBeMissing:
		logger.WithError(err).Warn(ctx, "Ignoring output artifact which does not exist in path")
		return false, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, argoerrs.Errorf(argoerrs.CodeNotFound, "required artifact '%s' not found at %s", art.Name, art.Path)
	case err != nil:
		return false, fmt.Errorf("failed to read artifact '%s' at %s: %w", art.Name, art.Path, err)
	}
	fileName, localArtPath, err := archiveLocalPath(ctx, art, copied, staging)
	if err != nil {
		return false, err
	}
	if localArtPath != copied {
		// archived, the copy only takes up space
		if err = os.RemoveAll(copied); err != nil {
			logger.WithError(err).Warn(ctx, "Failed to remove the copied output artifact")
		}
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

// copyOutputArtifact copies the file or directory at name in a plugin task's
// directory to dst, out of the plugins' reach. A missing name is reported as
// fs.ErrNotExist.
func (ae *AgentExecutor) copyOutputArtifact(nodeID, name, dst string, noSymlinks bool) error {
	dir, err := ae.openTask(nodeID)
	if err != nil {
		return err
	}
	defer dir.Close()
	if _, err = dir.Stat(name); err != nil {
		return err
	}
	return copyOut(dir, name, dst, noSymlinks)
}

// openTask opens a plugin task's directory in the workspace, which a plugin
// may have replaced with a symlink out of it.
func (ae *AgentExecutor) openTask(nodeID string) (*os.Root, error) {
	workspace, err := os.OpenRoot(ae.workspace)
	if err != nil {
		return nil, err
	}
	defer workspace.Close()
	return workspace.OpenRoot(nodeID)
}

// removeTask removes what a plugin task that is done leaves: the agent's
// staging directory for it, marker included, then the task's directory.
// Whatever a plugin running as another user made in that may not be
// removable; that goes with the agent pod.
func (ae *AgentExecutor) removeTask(ctx context.Context, nodeID string) {
	logger := logging.RequireLoggerFromContext(ctx)
	if err := os.RemoveAll(filepath.Join(ae.stagingDir, nodeID)); err != nil {
		logger.WithError(err).Warn(ctx, "Failed to remove the task's staging directory")
	}
	workspace, err := os.OpenRoot(ae.workspace)
	if err == nil {
		err = workspace.RemoveAll(nodeID)
		_ = workspace.Close()
	}
	if err != nil {
		logger.WithError(err).Warn(ctx, "Failed to remove the task's directory")
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

// taskPath returns where an artifact path of a plugin task is, relative to the
// task's directory, which stands for the root of the plugin's file system:
// ".." goes no higher than it.
func taskPath(artPath string) string {
	p := path.Clean("/" + artPath)[1:]
	if p == "" {
		return "."
	}
	return filepath.FromSlash(p)
}

// openNoBlock opens name in dir to read it, without waiting on a named pipe a
// plugin may have put there.
func openNoBlock(dir *os.Root, name string) (*os.File, error) {
	return dir.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

// mkdirShared makes the directory name in dir unless it exists, and opens it
// to every user.
func mkdirShared(dir *os.Root, name string) error {
	if err := dir.Mkdir(name, 0o777); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return chmodDir(dir, name, 0o777)
}

// mkdirAllShared makes the directory name in dir, with the parents it lacks,
// open to every user.
func mkdirAllShared(dir *os.Root, name string) error {
	if name == "." {
		return nil
	}
	p := ""
	for elem := range strings.SplitSeq(name, string(filepath.Separator)) {
		p = filepath.Join(p, elem)
		if err := mkdirShared(dir, p); err != nil {
			return err
		}
	}
	return nil
}

// chmodDir changes the mode of the directory name in dir, failing for
// anything else there.
func chmodDir(dir *os.Root, name string, perm fs.FileMode) error {
	f, err := openNoBlock(dir, name)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", name)
	}
	// unlike the mode Mkdir is given, this is not subject to the umask
	return f.Chmod(perm)
}

// copyIn copies the file or directory at src, out of the plugins' reach, to
// name in dir, keeping symlinks, and giving files and directories the mode
// that mode returns for them, given whether they are src itself and the mode
// they have.
func copyIn(src string, dir *os.Root, name string, mode func(top bool, mode fs.FileMode) fs.FileMode) error {
	type dirMode struct {
		name string
		perm fs.FileMode
	}
	var dirModes []dirMode
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(name, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch m := info.Mode(); {
		case m&fs.ModeSymlink != 0:
			var target string
			if target, err = os.Readlink(p); err != nil {
				return err
			}
			return dir.Symlink(target, dst)
		case m.IsDir():
			dirModes = append(dirModes, dirMode{dst, mode(p == src, m)})
			// for the agent to read the directory, and write to its copy,
			// until the files are in
			if err = os.Chmod(p, 0o700); err != nil {
				return err
			}
			return dir.Mkdir(dst, 0o700)
		case m.IsRegular():
			return copyFileIn(p, dir, dst, mode(p == src, m))
		default:
			return fmt.Errorf("%s is not a regular file, directory or symlink", p)
		}
	})
	if err != nil {
		return err
	}
	// the files are in once the directories in them are
	for _, d := range slices.Backward(dirModes) {
		if err = chmodDir(dir, d.name, d.perm); err != nil {
			return err
		}
	}
	return nil
}

// copyFileIn copies the file at src, out of the plugins' reach, to a new file
// name in dir with the mode perm.
func copyFileIn(src string, dir *os.Root, name string, perm fs.FileMode) error {
	// for the agent to read, whatever mode the file was loaded with
	if err := os.Chmod(src, 0o600); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// not following whatever a plugin may have put there since
	out, err := dir.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	return copyFile(out, in, perm)
}

// copyOut copies the file or directory at name in dir to dst, out of the
// plugins' reach, keeping modes as far as the agent can read the copy with
// them. A symlink at name is followed, within dir, as the plugin's own path
// would be. Under name, the agent copies the files and directories it lists,
// failing if one changes into anything else meanwhile, and the symlinks it
// lists as they are, unless noSymlinks: an artifact uploaded with no archive
// cannot hold them, as the artifact driver would follow them.
func copyOut(dir *os.Root, name, dst string, noSymlinks bool) error {
	return copyListedOut(dir, name, nil, dst, noSymlinks)
}

// copyListedOut copies name in dir as copyOut does. Unless listed is nil,
// name is an entry of a directory being copied, and listed what the agent
// found there. Opening name follows any symlink a plugin swapped in since:
// the file it opens must be the one listed, which a symlink to it, say once
// the plugin renamed it, still is.
func copyListedOut(dir *os.Root, name string, listed fs.FileInfo, dst string, noSymlinks bool) error {
	f, err := openNoBlock(dir, name)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if listed != nil && !os.SameFile(listed, info) {
		return argoerrs.Errorf(argoerrs.CodeBadRequest, "%s changed while the agent copied it", pluginPath(name))
	}
	mode := info.Mode()
	switch {
	case mode.IsRegular():
		var out *os.File
		if out, err = os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600); err != nil {
			return err
		}
		// archived with its mode, which the agent must be able to read it with
		return copyFile(out, f, mode.Perm()|0o400)
	case !mode.IsDir():
		return argoerrs.Errorf(argoerrs.CodeBadRequest, "%s is not a regular file, directory or symlink", pluginPath(name))
	}
	if err = os.Mkdir(dst, 0o700); err != nil {
		return err
	}
	entries, err := f.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, e := range entries {
		child, childDst := filepath.Join(name, e.Name()), filepath.Join(dst, e.Name())
		var childInfo fs.FileInfo
		if childInfo, err = dir.Lstat(child); err != nil {
			return err
		}
		switch {
		case childInfo.Mode()&fs.ModeSymlink == 0:
			err = copyListedOut(dir, child, childInfo, childDst, noSymlinks)
		case noSymlinks:
			err = argoerrs.Errorf(argoerrs.CodeBadRequest, "%s is a symlink, which an artifact with archive none cannot hold", pluginPath(child))
		default:
			var target string
			// fails for anything a plugin swapped the symlink for since
			if target, err = dir.Readlink(child); err == nil {
				err = os.Symlink(target, childDst)
			}
		}
		if err != nil {
			return err
		}
	}
	// the agent, which is not root, can only remove a directory's files if it
	// can write to it
	return os.Chmod(dst, mode.Perm()|0o700)
}

// copyFile copies in to out, gives out the mode perm and closes it.
func copyFile(out, in *os.File, perm fs.FileMode) error {
	_, err := io.Copy(out, in)
	if err == nil {
		// unlike the mode out was created with, this is not subject to the umask
		err = out.Chmod(perm)
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

// pluginPath returns the path a plugin's template gives for name in a task's
// directory.
func pluginPath(name string) string {
	return "/" + filepath.ToSlash(name)
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
	switch {
	case apierrors.IsForbidden(argoerrs.Cause(err)):
		return "", fmt.Errorf("secret %q is not mounted in the agent pod, and the agent's service account may not get it from the Kubernetes API: grant it a Role allowing get on secrets in namespace %s: %w", name, r.namespace, err)
	case err != nil:
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
	return s != "." && filepath.IsLocal(s) && filepath.Base(s) == s
}
