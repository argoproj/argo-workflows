package executor

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	retryutil "k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-workflows/v4/util/file"

	argoerrs "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoprojv1 "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/archive"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v4/util/wait"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts"
	artifactcommon "github.com/argoproj/argo-workflows/v4/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	executorretry "github.com/argoproj/argo-workflows/v4/workflow/executor/retry"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

const (
	// This directory temporarily stores the tarballs of the artifacts before uploading
	tempOutArtDir = "/tmp/argo/outputs/artifacts"
)

// WorkflowExecutor is program which runs as the init/wait container
type WorkflowExecutor struct {
	PodName             string
	podUID              types.UID
	workflow            string
	workflowUID         types.UID
	nodeID              string
	Template            wfv1.Template
	IncludeScriptOutput bool
	Deadline            time.Time
	ClientSet           kubernetes.Interface
	taskResultClient    argoprojv1.WorkflowTaskResultInterface
	RESTClient          rest.Interface
	Namespace           string
	RuntimeExecutor     ContainerRuntimeExecutor
	Tracing             *tracing.Tracing

	// memoized configmaps
	memoizedConfigMaps map[string]string
	// memoized secrets
	memoizedSecrets map[string][]byte
	// list of errors that occurred during execution.
	// the first of these is used as the overall message of the node
	errors []error

	// current progress which is synced every `annotationPatchTickDuration` to the pods annotations.
	progress wfv1.Progress

	annotationPatchTickDuration  time.Duration
	readProgressFileTickDuration time.Duration

	// flag to indicate if the task result was created
	taskResultCreated bool
}

type Initializer interface {
	Init(tmpl wfv1.Template) error
}

// ContainerRuntimeExecutor is the interface for interacting with a container runtime
type ContainerRuntimeExecutor interface {
	// GetFileContents returns the file contents of a file in a container as a string
	GetFileContents(containerName string, sourcePath string) (string, error)

	// CopyFile copies a source file in a container to a local path
	CopyFile(ctx context.Context, containerName, sourcePath, destPath string, compressionLevel int) error

	// GetOutputStream returns the entirety of the container output as a io.Reader
	// Used to capture script results as an output parameter, and to archive container logs
	GetOutputStream(ctx context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error)

	// Wait waits for the container to complete.
	Wait(ctx context.Context, containerNames []string) error

	// Kill a list of containers first with a SIGTERM then with a SIGKILL after a grace period
	Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error
}

// WorkflowName returns the name of the workflow being executed
func (we *WorkflowExecutor) WorkflowName() string {
	return we.workflow
}

// NewExecutor instantiates a new workflow executor
func NewExecutor(
	ctx context.Context,
	clientset kubernetes.Interface,
	taskResultClient argoprojv1.WorkflowTaskResultInterface,
	restClient rest.Interface,
	podName string,
	podUID types.UID,
	workflow string,
	workflowUID types.UID,
	nodeID, namespace string,
	cre ContainerRuntimeExecutor,
	template wfv1.Template,
	includeScriptOutput bool,
	deadline time.Time,
	annotationPatchTickDuration, readProgressFileTickDuration time.Duration,
) (WorkflowExecutor, error) {
	retry := executorretry.ExecutorRetry(ctx)
	logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
		"Steps":    retry.Steps,
		"Duration": retry.Duration,
		"Factor":   retry.Factor,
		"Jitter":   retry.Jitter,
	}).Info(ctx, "Using executor retry strategy")
	tracing, err := tracing.New(ctx, `argoexec`)
	return WorkflowExecutor{
		PodName:                      podName,
		podUID:                       podUID,
		workflow:                     workflow,
		workflowUID:                  workflowUID,
		nodeID:                       nodeID,
		ClientSet:                    clientset,
		taskResultClient:             taskResultClient,
		RESTClient:                   restClient,
		Namespace:                    namespace,
		RuntimeExecutor:              cre,
		Template:                     template,
		IncludeScriptOutput:          includeScriptOutput,
		Deadline:                     deadline,
		Tracing:                      tracing,
		memoizedConfigMaps:           map[string]string{},
		memoizedSecrets:              map[string][]byte{},
		errors:                       []error{},
		annotationPatchTickDuration:  annotationPatchTickDuration,
		readProgressFileTickDuration: readProgressFileTickDuration,
	}, err
}

// HandleError is a helper to annotate the pod with the error message upon a unexpected executor panic or error.
// Usage: defer we.HandleError(ctx)()
//
//nolint:revive // recover is inside returned closure that gets deferred by caller
func (we *WorkflowExecutor) HandleError(ctx context.Context) func() {
	return func() {
		if r := recover(); r != nil {
			util.WriteTerminateMessage(fmt.Sprintf("%v", r))
			logging.RequireLoggerFromContext(ctx).WithFatal().WithFields(logging.Fields{
				"error": r,
				"stack": debug.Stack(),
			}).Error(ctx, "executor panic")
		} else if len(we.errors) > 0 {
			util.WriteTerminateMessage(we.errors[0].Error())
		}
	}
}

func (we *WorkflowExecutor) LoadArtifactsWithoutPlugins(ctx context.Context) error {
	return we.loadArtifacts(ctx, "")
}

func (we *WorkflowExecutor) LoadArtifactsFromPlugin(ctx context.Context, pluginName wfv1.ArtifactPluginName) error {
	return we.loadArtifacts(ctx, pluginName)
}

// loadArtifacts loads artifacts from location to a container path
// pluginName is the name of the plugin to load artifacts from, only one plugin can be used at a time
func (we *WorkflowExecutor) loadArtifacts(ctx context.Context, pluginName wfv1.ArtifactPluginName) error {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"pluginName": pluginName}).Info(ctx, "Start loading input artifacts...")
	for _, art := range we.Template.Inputs.Artifacts {
		err := we.loadArtifact(ctx, pluginName, art)
		if err != nil {
			return err
		}
	}
	return nil
}

func (we *WorkflowExecutor) loadArtifact(ctx context.Context, pluginName wfv1.ArtifactPluginName, art wfv1.Artifact) error {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("name", art.Name).Info(ctx, "Downloading artifact")

	if !art.HasLocationOrKey() {
		if art.Optional {
			logger.WithField("name", art.Name).Warn(ctx, "Ignoring optional artifact which was not supplied")
			return nil
		}
		return argoerrs.Errorf(argoerrs.CodeNotFound, "required artifact '%s' not supplied", art.Name)
	}
	err := art.CleanPath()
	if err != nil {
		return err
	}
	driverArt, err := we.newDriverArt(&art)
	if err != nil {
		return fmt.Errorf("failed to load artifact '%s': %w", art.Name, err)
	}
	switch pluginName {
	// If no plugin is specified only load non-plugin artifacts
	case "":
		if driverArt.Plugin != nil {
			logger.Info(ctx, "Skipping artifact that is from a plugin")
			return nil
		}
		// If a plugin is specified only load artifacts from that plugin
	default:
		if driverArt.Plugin == nil || driverArt.Plugin.Name != pluginName {
			logger.WithFields(logging.Fields{"name": driverArt.Name, "plugin": driverArt.Plugin}).Info(ctx, "Skipping artifact that is not from the specified plugin")
			return nil
		}
	}

	artDriver, err := we.InitDriver(ctx, driverArt)
	if err != nil {
		return err
	}
	// Determine the file path of where to load the artifact
	var artPath string
	mnt := common.FindOverlappingVolume(&we.Template, art.Path)
	if mnt == nil {
		artPath = path.Join(common.ExecutorArtifactBaseDir, art.Name)
	} else {
		// If we get here, it means the input artifact path overlaps with a user-specified
		// volumeMount in the container. Because we also implement input artifacts as volume
		// mounts, we need to load the artifact into the user specified volume mount,
		// as opposed to the `input-artifacts` volume that is an implementation detail
		// unbeknownst to the user.
		logger.WithFields(logging.Fields{"path": art.Path, "mountPath": mnt.MountPath}).Info(ctx, "Specified artifact path overlaps with volume mount, extracting to volume mount")
		artPath = path.Join(common.ExecutorMainFilesystemDir, art.Path)
	}

	// The artifact is downloaded to a temporary location, after which we determine if
	// the file is a tarball or not. If it is, it is first extracted then renamed to
	// the desired location. If not, it is simply renamed to the location.
	tempArtPath := artPath + ".tmp"
	// Ensure parent directory exist, create if missing
	tempArtDir := filepath.Dir(tempArtPath)
	if err := os.MkdirAll(tempArtDir, 0o700); err != nil {
		return fmt.Errorf("failed to create artifact temporary parent directory %s: %w", tempArtDir, err)
	}
	err = artDriver.Load(ctx, driverArt, tempArtPath)
	if err != nil {
		if art.Optional && argoerrs.IsCode(argoerrs.CodeNotFound, err) {
			logger.WithField("name", art.Name).Info(ctx, "Skipping optional input artifact that was not found")
			return nil
		}
		return fmt.Errorf("artifact %s failed to load: %w", art.Name, err)
	}

	err = we.unarchiveArtifact(ctx, art, tempArtPath, artPath)
	if err != nil {
		return err
	}

	logger.WithField("path", artPath).Info(ctx, "Successfully download file")
	if art.Mode != nil {
		err = chmod(artPath, *art.Mode, art.RecurseMode)
		if err != nil {
			return err
		}
	} else if driverArt.Plugin != nil {
		// For plugin artifacts without explicit mode, ensure the file is writable
		// by setting mode to 0666 so the main container can read/write it
		err = chmod(artPath, 0666, art.RecurseMode)
		if err != nil {
			logger.WithError(err).Error(ctx, "Failed to chmod plugin artifact")
			return err
		}
	}
	return nil
}

func (we *WorkflowExecutor) unarchiveArtifact(ctx context.Context, art wfv1.Artifact, tempArtPath, artPath string) error {
	isTar := false
	isZip := false
	var err error

	switch {
	case art.GetArchive().None != nil:
		// explicitly not a tar
		isTar = false
		isZip = false
	case art.GetArchive().Tar != nil:
		// explicitly a tar
		isTar = true
	case art.GetArchive().Zip != nil:
		// explicitly a zip
		isZip = true
	default:
		// auto-detect if tarball
		// (don't try to autodetect zip files for backwards compatibility)
		isTar, err = isTarball(ctx, tempArtPath)
		if err != nil {
			return err
		}
	}

	switch {
	case isTar:
		err = untar(tempArtPath, artPath)
		_ = os.Remove(tempArtPath)
	case isZip:
		err = unzip(ctx, tempArtPath, artPath)
		_ = os.Remove(tempArtPath)
	default:
		err = os.Rename(tempArtPath, artPath)
	}
	return err
}

// StageFiles will create any files required by script/resource templates
func (we *WorkflowExecutor) StageFiles(ctx context.Context) error {
	var filePath string
	var body []byte
	logger := logging.RequireLoggerFromContext(ctx)
	mode := os.FileMode(0o644)
	switch we.Template.GetType() {
	case wfv1.TemplateTypeScript:
		logger.WithField("path", common.ExecutorScriptSourcePath).Info(ctx, "Loading script source")
		filePath = common.ExecutorScriptSourcePath
		body = []byte(we.Template.Script.Source)
		mode = os.FileMode(0o755)
	case wfv1.TemplateTypeResource:
		if we.Template.Resource.ManifestFrom != nil && we.Template.Resource.ManifestFrom.Artifact != nil {
			logger.WithField("name", we.Template.Resource.ManifestFrom.Artifact.Name).Info(ctx, "manifest already staged")
			return nil
		}
		logger.WithField("path", common.ExecutorResourceManifestPath).Info(ctx, "Loading manifest")
		filePath = common.ExecutorResourceManifestPath
		body = []byte(we.Template.Resource.Manifest)
	default:
		return nil
	}
	err := os.WriteFile(filePath, body, mode)
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}
	return nil
}

// SaveArtifacts uploads artifacts to the archive location
func (we *WorkflowExecutor) SaveArtifacts(ctx context.Context) (wfv1.Artifacts, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	artifacts := wfv1.Artifacts{}
	if len(we.Template.Outputs.Artifacts) == 0 {
		logger.Info(ctx, "No output artifacts")
		return artifacts, nil
	}

	logger.Info(ctx, "Saving output artifacts")
	err := os.MkdirAll(tempOutArtDir, os.ModePerm)
	if err != nil {
		return artifacts, argoerrs.InternalWrapError(err)
	}

	aggregateError := ""
	for _, art := range we.Template.Outputs.Artifacts {
		saved, err := we.saveArtifact(ctx, common.MainContainerName, &art)
		if err != nil {
			aggregateError += err.Error() + "; "
		}
		if saved {
			artifacts = append(artifacts, art)
		}
	}
	if aggregateError == "" {
		return artifacts, nil
	}
	return artifacts, errors.New(aggregateError)
}

// save artifact
// return whether artifact was in fact saved, and if there was an error
func (we *WorkflowExecutor) saveArtifact(ctx context.Context, containerName string, art *wfv1.Artifact) (bool, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	// Determine the file path of where to find the artifact
	err := art.CleanPath()
	if err != nil {
		return false, err
	}
	fileName, localArtPath, err := we.stageArchiveFile(ctx, containerName, art)
	if err != nil {
		if art.Optional && argoerrs.IsCode(argoerrs.CodeNotFound, err) {
			logger.WithField("name", art.Name).WithField("path", art.Path).WithError(err).Warn(ctx, "Ignoring optional artifact which does not exist in path")
			return false, nil
		}
		return false, err
	}
	fi, err := os.Stat(localArtPath)
	if err != nil {
		return false, err
	}
	size := fi.Size()
	if size == 0 {
		logger.WithField("path", localArtPath).Warn(ctx, "The file is empty. It may not be uploaded successfully depending on the artifact driver")
	}
	err = we.saveArtifactFromFile(ctx, art, fileName, localArtPath)
	return err == nil, err
}

// fileBase is probably path.Base(filePath), but can be something else
func (we *WorkflowExecutor) saveArtifactFromFile(ctx context.Context, art *wfv1.Artifact, fileName, localArtPath string) error {
	if !art.HasKey() {
		key, err := we.Template.ArchiveLocation.GetKey()
		if err != nil {
			return err
		}
		artLocation, err := we.Template.ArchiveLocation.Get()
		if err != nil {
			return err
		}
		if err = art.SetType(artLocation); err != nil {
			return err
		}
		if err := art.SetKey(path.Join(key, fileName)); err != nil {
			return err
		}
	}
	driverArt, err := we.newDriverArt(art)
	if err != nil {
		return err
	}
	artDriver, err := we.InitDriver(ctx, driverArt)
	if err != nil {
		return err
	}
	err = artDriver.Save(ctx, localArtPath, driverArt)
	if err != nil {
		return err
	}
	we.maybeDeleteLocalArtPath(ctx, localArtPath)
	logging.RequireLoggerFromContext(ctx).WithField("path", localArtPath).Info(ctx, "Successfully saved file")
	return nil
}

func (we *WorkflowExecutor) maybeDeleteLocalArtPath(ctx context.Context, localArtPath string) {
	if os.Getenv("REMOVE_LOCAL_ART_PATH") == "true" {
		logger := logging.RequireLoggerFromContext(ctx)
		logger.WithField("localArtPath", localArtPath).Info(ctx, "deleting local artifact")
		// remove is best effort (the container will go away anyways).
		// we just want reduce peak space usage
		err := os.Remove(localArtPath)
		if err != nil {
			logger.WithField("path", localArtPath).WithError(err).Warn(ctx, "Failed to remove")
		}
	} else {
		logging.RequireLoggerFromContext(ctx).WithField("localArtPath", localArtPath).Info(ctx, "not deleting local artifact")
	}
}

// stageArchiveFile stages a path in a container for archiving from the wait sidecar.
// Returns a filename and a local path for the upload.
// The filename is incorporated into the final path when uploading it to the artifact repo.
// The local path is the final staging location of the file (or directory) which we will pass
// to the SaveArtifacts call and may be a directory or file.
func (we *WorkflowExecutor) stageArchiveFile(ctx context.Context, containerName string, art *wfv1.Artifact) (string, string, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("name", art.Name).Info(ctx, "Staging artifact")
	strategy := art.Archive
	if strategy == nil {
		// If no strategy is specified, default to the tar strategy
		strategy = &wfv1.ArchiveStrategy{
			Tar: &wfv1.TarStrategy{},
		}
	}
	compressionLevel := gzip.NoCompression
	if strategy.Tar != nil {
		if l := strategy.Tar.CompressionLevel; l != nil {
			compressionLevel = int(*l)
		} else {
			compressionLevel = gzip.DefaultCompression
		}
	}

	if !we.isBaseImagePath(art.Path) {
		// If we get here, we are uploading an artifact from a mirrored volume mount which the wait
		// sidecar has direct access to. We can upload directly from the shared volume mount,
		// instead of copying it from the container.
		mountedArtPath := filepath.Join(common.ExecutorMainFilesystemDir, art.Path)
		logger.WithFields(logging.Fields{"path": art.Path, "mountedArtPath": mountedArtPath}).Info(ctx, "Staging from mirrored volume mount")
		if strategy.None != nil {
			fileName := filepath.Base(art.Path)
			logger.WithField("fileName", fileName).Info(ctx, "No compression strategy needed, staging skipped")
			if !file.Exists(mountedArtPath) {
				return "", "", argoerrs.Errorf(argoerrs.CodeNotFound, "%s no such file or directory", art.Path)
			}
			return fileName, mountedArtPath, nil
		}
		if strategy.Zip != nil {
			fileName := fmt.Sprintf("%s.zip", art.Name)
			localArtPath := filepath.Join(tempOutArtDir, fileName)
			f, err := os.Create(localArtPath)
			if err != nil {
				return "", "", argoerrs.InternalWrapError(err)
			}
			zw := zip.NewWriter(f)
			defer zw.Close()
			err = archive.ZipToWriter(ctx, mountedArtPath, zw)
			if err != nil {
				return "", "", err
			}
			logger.WithFields(logging.Fields{"path": art.Path, "mountedArtPath": mountedArtPath}).Info(ctx, "Successfully staged from mirrored volume mount")
			return fileName, localArtPath, nil
		}
		fileName := fmt.Sprintf("%s.tgz", art.Name)
		localArtPath := filepath.Join(tempOutArtDir, fileName)
		f, err := os.Create(localArtPath)
		if err != nil {
			return "", "", argoerrs.InternalWrapError(err)
		}
		w := bufio.NewWriter(f)
		err = archive.TarGzToWriter(ctx, mountedArtPath, compressionLevel, w)
		if err != nil {
			return "", "", err
		}
		logger.WithFields(logging.Fields{"path": art.Path, "mountedArtPath": mountedArtPath}).Info(ctx, "Successfully staged from mirrored volume mount")
		return fileName, localArtPath, nil
	}

	fileName := fmt.Sprintf("%s.tgz", art.Name)
	localArtPath := filepath.Join(tempOutArtDir, fileName)
	logger.WithFields(logging.Fields{"path": art.Path, "localArtPath": localArtPath}).Info(ctx, "Copying from container base image layer")

	err := we.RuntimeExecutor.CopyFile(ctx, containerName, art.Path, localArtPath, compressionLevel)
	if err != nil {
		return "", "", err
	}
	if strategy.Tar != nil {
		// NOTE we already tar gzip the file in the executor. So this is a noop.
		return fileName, localArtPath, nil
	}
	// localArtPath now points to a .tgz file, and the archive strategy is *not* tar. We need to untar it
	logger.WithField("path", localArtPath).Info(ctx, "Untarring archive before upload")
	unarchivedArtPath := path.Join(filepath.Dir(localArtPath), art.Name)
	err = untar(localArtPath, unarchivedArtPath)
	if err != nil {
		return "", "", err
	}
	// Delete the tarball
	err = os.Remove(localArtPath)
	if err != nil {
		return "", "", argoerrs.InternalWrapError(err)
	}
	isDir, err := file.IsDirectory(unarchivedArtPath)
	if err != nil {
		return "", "", argoerrs.InternalWrapError(err)
	}
	fileName = filepath.Base(art.Path)
	if isDir {
		localArtPath = unarchivedArtPath
	} else {
		// If we are uploading a single file, we need to preserve original filename so that
		// 1. minio client can infer its mime-type, based on file extension
		// 2. the original filename is incorporated into the final path
		localArtPath = path.Join(tempOutArtDir, fileName)
		err = os.Rename(unarchivedArtPath, localArtPath)
		if err != nil {
			return "", "", argoerrs.InternalWrapError(err)
		}
	}
	// In the future, if we were to support other compression formats (e.g. bzip2) or options
	// the logic would go here, and compression would be moved out of the executors
	if strategy.Zip != nil {
		fileName = fmt.Sprintf("%s.zip", art.Name)
		localArtPath = filepath.Join(tempOutArtDir, fileName)
		f, err := os.Create(localArtPath)
		if err != nil {
			return "", "", argoerrs.InternalWrapError(err)
		}
		zw := zip.NewWriter(f)
		defer zw.Close()
		err = archive.ZipToWriter(ctx, unarchivedArtPath, zw)
		if err != nil {
			return "", "", err
		}
		logger.WithFields(logging.Fields{"unarchivedArtPath": unarchivedArtPath, "localArtPath": localArtPath}).Info(ctx, "Successfully zipped")
		return fileName, localArtPath, nil
	}
	return fileName, localArtPath, nil
}

// isBaseImagePath checks if the given artifact path resides in the base image layer of the container
// versus a shared volume mount between the wait and main container
func (we *WorkflowExecutor) isBaseImagePath(path string) bool {
	// first check if path overlaps with a user-specified volumeMount
	if common.FindOverlappingVolume(&we.Template, path) != nil {
		return false
	}
	// next check if path overlaps with a shared input-artifact emptyDir mounted by argo
	for _, inArt := range we.Template.Inputs.Artifacts {
		if path == inArt.Path {
			// The input artifact may have been optional and not supplied. If this is the case, the file won't exist on
			// the input artifact volume. Since this function was called, we know that we want to use this path as an
			// output artifact, so we should look for it in the base image path.
			if inArt.Optional && !inArt.HasLocationOrKey() {
				return true
			}
			return false
		}
		if strings.HasPrefix(path, inArt.Path+"/") {
			return false
		}
	}
	return true
}

// SaveParameters will save the content in the specified file path as output parameter value
func (we *WorkflowExecutor) SaveParameters(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	if len(we.Template.Outputs.Parameters) == 0 {
		logger.Info(ctx, "No output parameters")
		return nil
	}
	logger.Info(ctx, "Saving output parameters")
	for i, param := range we.Template.Outputs.Parameters {
		logger.WithField("name", param.Name).Info(ctx, "Saving path output parameter")
		// Determine the file path of where to find the parameter
		if param.ValueFrom == nil || param.ValueFrom.Path == "" {
			continue
		}

		var output *wfv1.AnyString
		if we.isBaseImagePath(param.ValueFrom.Path) {
			logger.WithField("path", param.ValueFrom.Path).Info(ctx, "Copying from base image layer")
			fileContents, err := we.RuntimeExecutor.GetFileContents(common.MainContainerName, param.ValueFrom.Path)
			if err != nil {
				// We have a default value to use instead of returning an error
				if param.ValueFrom.Default == nil {
					return err
				}
				output = param.ValueFrom.Default
			} else {
				output = wfv1.AnyStringPtr(fileContents)
			}
		} else {
			logger.WithField("path", param.ValueFrom.Path).Info(ctx, "Copying from volume mount")
			mountedPath := filepath.Join(common.ExecutorMainFilesystemDir, param.ValueFrom.Path)
			data, err := os.ReadFile(filepath.Clean(mountedPath))
			if err != nil {
				// We have a default value to use instead of returning an error
				if param.ValueFrom.Default == nil {
					return err
				}
				output = param.ValueFrom.Default
			} else {
				output = wfv1.AnyStringPtr(string(data))
			}
		}

		// Trims off a single newline for user convenience
		output = wfv1.AnyStringPtr(strings.TrimSuffix(output.String(), "\n"))
		we.Template.Outputs.Parameters[i].Value = output
		logger.WithField("name", param.Name).Info(ctx, "Successfully saved output parameter")
	}
	return nil
}

func (we *WorkflowExecutor) SaveLogs(ctx context.Context) []wfv1.Artifact {
	var logArtifacts []wfv1.Artifact
	tempLogsDir := "/tmp/argo/outputs/logs"

	if we.Template.SaveLogsAsArtifact() {
		err := os.MkdirAll(tempLogsDir, os.ModePerm)
		if err != nil {
			we.AddError(ctx, argoerrs.InternalWrapError(err))
		}

		containerNames := we.Template.GetMainContainerNames()
		logArtifacts = make([]wfv1.Artifact, 0)

		for _, containerName := range containerNames {
			// Saving logs
			art, err := we.saveContainerLogs(ctx, tempLogsDir, containerName)
			if err != nil {
				we.AddError(ctx, err)
			} else {
				logArtifacts = append(logArtifacts, *art)
			}
		}
	}

	return logArtifacts
}

// saveContainerLogs saves a single container's log into a file
func (we *WorkflowExecutor) saveContainerLogs(ctx context.Context, tempLogsDir, containerName string) (*wfv1.Artifact, error) {
	fileName := containerName + ".log"
	filePath := path.Join(tempLogsDir, fileName)
	err := we.saveLogToFile(ctx, containerName, filePath)
	if err != nil {
		return nil, err
	}

	art := &wfv1.Artifact{Name: containerName + "-logs"}
	err = we.saveArtifactFromFile(ctx, art, fileName, filePath)
	if err != nil {
		return nil, err
	}

	return art, nil
}

// GetSecret will retrieve the Secrets from VolumeMount
func (we *WorkflowExecutor) GetSecret(ctx context.Context, accessKeyName string, accessKey string) (string, error) {
	file, err := os.ReadFile(filepath.Clean(filepath.Join(common.SecretVolMountPath, accessKeyName, accessKey)))
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// saveLogToFile saves the entire log output of a container to a local file
func (we *WorkflowExecutor) saveLogToFile(ctx context.Context, containerName, path string) error {
	outFile, err := os.Create(path)
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}
	defer func() { _ = outFile.Close() }()
	reader, err := we.RuntimeExecutor.GetOutputStream(ctx, containerName, true)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	_, err = io.Copy(outFile, reader)
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}
	return nil
}

func (we *WorkflowExecutor) newDriverArt(art *wfv1.Artifact) (*wfv1.Artifact, error) {
	driverArt := art.DeepCopy()
	err := driverArt.Relocate(we.Template.ArchiveLocation)
	return driverArt, err
}

// InitDriver initializes an instance of an artifact driver
func (we *WorkflowExecutor) InitDriver(ctx context.Context, art *wfv1.Artifact) (artifactcommon.ArtifactDriver, error) {
	driver, err := artifacts.NewDriver(ctx, art, we)
	if errors.Is(err, artifacts.ErrUnsupportedDriver) {
		return nil, argoerrs.Errorf(argoerrs.CodeBadRequest, "Unsupported artifact driver for %s", art.Name)
	}
	return driver, err
}

// GetConfigMapKey retrieves a configmap value and memoizes the result
func (we *WorkflowExecutor) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	namespace := we.Namespace
	cachedKey := fmt.Sprintf("%s/%s/%s", namespace, name, key)
	if val, ok := we.memoizedConfigMaps[cachedKey]; ok {
		return val, nil
	}
	configmapsIf := we.ClientSet.CoreV1().ConfigMaps(namespace)
	var configmap *apiv1.ConfigMap
	err := waitutil.Backoff(retry.DefaultRetry(ctx), func() (bool, error) {
		var err error
		configmap, err = configmapsIf.Get(ctx, name, metav1.GetOptions{})
		return !errorsutil.IsTransientErr(ctx, err), err
	})
	if err != nil {
		return "", argoerrs.InternalWrapError(err)
	}
	// memoize all keys in the configmap since it's highly likely we will need to get a
	// subsequent key in the configmap (e.g. username + password) and we can save an API call
	for k, v := range configmap.Data {
		we.memoizedConfigMaps[fmt.Sprintf("%s/%s/%s", namespace, name, k)] = v
	}
	val, ok := we.memoizedConfigMaps[cachedKey]
	if !ok {
		return "", argoerrs.Errorf(argoerrs.CodeBadRequest, "configmap '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// GetTerminationGracePeriodDuration returns the terminationGracePeriodSeconds of podSpec in Time.Duration format
func GetTerminationGracePeriodDuration() time.Duration {
	x, _ := strconv.ParseInt(os.Getenv(common.EnvVarTerminationGracePeriodSeconds), 10, 64)
	if x > 0 {
		return time.Duration(x) * time.Second
	}
	return 30 * time.Second
}

// CaptureScriptResult will add the stdout of a script template as output result
func (we *WorkflowExecutor) CaptureScriptResult(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	if !we.IncludeScriptOutput {
		logger.Info(ctx, "No Script output reference in workflow. Capturing script output ignored")
		return nil
	}
	if !we.Template.HasOutput() {
		logger.Info(ctx, "Template type is neither of Script, Container, or Pod. Capturing script output ignored")
		return nil
	}
	logger.Info(ctx, "Capturing script output")
	reader, err := we.RuntimeExecutor.GetOutputStream(ctx, common.MainContainerName, false)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}
	out := string(bytes)
	// Trims off a single newline for user convenience
	outputLen := len(out)
	if outputLen > 0 && out[outputLen-1] == '\n' {
		out = out[0 : outputLen-1]
	}

	const maxAnnotationSize int = 256 * (1 << 10) // 256 kB
	// A character in a string is a byte
	if len(out) > maxAnnotationSize {
		logger.Warn(ctx, "Output is larger than the maximum allowed size of 256 kB, only the last 256 kB were saved")
		out = out[len(out)-maxAnnotationSize:]
	}

	we.Template.Outputs.Result = &out
	return nil
}

// FinalizeOutput adds a label or annotation to denote that outputs have completed reporting.
func (we *WorkflowExecutor) FinalizeOutput(ctx context.Context) {
	logger := logging.RequireLoggerFromContext(ctx)
	var count uint64
	err := retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
		err := we.patchTaskResultLabels(ctx, map[string]string{
			common.LabelKeyReportOutputsCompleted: "true",
		})
		if apierr.IsForbidden(err) || apierr.IsNotFound(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
	if err != nil {
		we.AddError(ctx, err)
	}
}

func (we *WorkflowExecutor) InitializeOutput(ctx context.Context) {
	logger := logging.RequireLoggerFromContext(ctx)
	var count uint64
	err := retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
		err := we.upsertTaskResult(ctx, wfv1.NodeResult{})
		if apierr.IsForbidden(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
	if err != nil {
		we.AddError(ctx, err)
	}
}

// ReportOutputs updates the WorkflowTaskResult (or falls back to annotate the Pod)
func (we *WorkflowExecutor) ReportOutputs(ctx context.Context, artifacts []wfv1.Artifact) error {
	outputs := we.Template.Outputs.DeepCopy()
	outputs.Artifacts = artifacts
	return we.reportResult(ctx, wfv1.NodeResult{Outputs: outputs})
}

// ReportOutputsLogs updates the WorkflowTaskResult log fields
func (we *WorkflowExecutor) ReportOutputsLogs(ctx context.Context) error {
	var outputs wfv1.Outputs
	artifacts := wfv1.Artifacts{}
	logArtifacts := we.SaveLogs(ctx)
	artifacts = append(artifacts, logArtifacts...)
	outputs.Artifacts = artifacts
	return we.reportResult(ctx, wfv1.NodeResult{Outputs: &outputs})
}

func (we *WorkflowExecutor) reportResult(ctx context.Context, result wfv1.NodeResult) error {
	var count uint64 // used to avoid spamming with these messages
	logger := logging.RequireLoggerFromContext(ctx)
	return retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2.0,
		Jitter:   0.2,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
		err := we.upsertTaskResult(ctx, result)
		if apierr.IsForbidden(err) {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		} else if err != nil && count%20 == 0 {
			logger.WithError(err).WithField("attempt", count).Warn(ctx, "failed to patch task result")
		}
		count++
		return err
	})
}

// AddError adds an error to the list of encountered errors during execution
func (we *WorkflowExecutor) AddError(ctx context.Context, err error) {
	logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "executor error")
	we.errors = append(we.errors, err)
}

// HasError return the first error if exist
func (we *WorkflowExecutor) HasError() error {
	if len(we.errors) > 0 {
		return we.errors[0]
	}
	return nil
}

// AddAnnotation adds an annotation to the workflow pod
func (we *WorkflowExecutor) AddAnnotation(ctx context.Context, key, value string) error {
	data, err := json.Marshal(map[string]any{"metadata": metav1.ObjectMeta{
		Annotations: map[string]string{
			key: value,
		},
	}})
	if err != nil {
		return err
	}
	_, err = we.ClientSet.CoreV1().Pods(we.Namespace).Patch(ctx, we.PodName, types.MergePatchType, data, metav1.PatchOptions{})
	return err
}

// isTarball returns whether or not the file is a tarball
func isTarball(ctx context.Context, filePath string) (bool, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("path", filePath).Info(ctx, "Detecting if file is a tarball")
	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.WithFatal().WithField("path", filePath).WithError(err).Error(ctx, "Error closing file")
		}
	}()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return false, nil
	}
	defer gzr.Close()
	tarr := tar.NewReader(gzr)
	_, err = tarr.Next()
	return err == nil, nil
}

// untar extracts a tarball to a temporary directory,
// renaming it to the desired location
func untar(tarPath string, destPath string) error {
	decompressor := func(src string, dest string) error {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		gzr, err := file.GetGzipReader(f)
		if err != nil {
			return err
		}
		defer gzr.Close()
		tr := tar.NewReader(gzr)
		for {
			header, err := tr.Next()
			switch {
			case errors.Is(err, io.EOF):
				return nil
			case err != nil:
				return err
			case header == nil:
				continue
			}
			target := filepath.Join(dest, filepath.Clean(header.Name))
			if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
				return fmt.Errorf("illegal file path: %s", header.Name)
			}
			switch header.Typeflag {
			case tar.TypeSymlink:
				// Validate symlink target before creating it
				linkTarget := header.Linkname
				if !filepath.IsAbs(linkTarget) {
					linkTarget = filepath.Join(filepath.Dir(target), header.Linkname)
				}
				if !strings.HasPrefix(filepath.Clean(linkTarget), filepath.Clean(dest)+string(os.PathSeparator)) {
					return fmt.Errorf("illegal symlink target: %s -> %s", header.Name, header.Linkname)
				}
				// Create parent directory if needed
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				err := os.Symlink(header.Linkname, target)
				if err != nil {
					return err
				}
			case tar.TypeDir:
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
			case tar.TypeReg:
				// Before writing the file, check if the parent directory resolves outside dest
				parentDir := filepath.Dir(target)

				// Resolve the destination directory
				resolvedDest, err := filepath.EvalSymlinks(dest)
				if err != nil {
					return err
				}

				// Check if parent exists and if so, verify it doesn't resolve outside dest
				if _, err := os.Lstat(parentDir); err == nil {
					// Parent exists, resolve it to check for symlink traversal
					resolvedParent, err := filepath.EvalSymlinks(parentDir)
					if err != nil {
						return err
					}
					// Check if resolved parent is outside dest
					if !strings.HasPrefix(resolvedParent+string(os.PathSeparator), resolvedDest+string(os.PathSeparator)) && resolvedParent != resolvedDest {
						return fmt.Errorf("illegal file path after symlink resolution: %s resolves outside destination", header.Name)
					}
				} else if !os.IsNotExist(err) {
					return err
				} else {
					// Parent doesn't exist, create it
					if err := os.MkdirAll(parentDir, 0o755); err != nil {
						return err
					}
				}

				f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				if _, err := io.Copy(f, tr); err != nil {
					return err
				}
				if err := f.Close(); err != nil {
					return err
				}
				if err := os.Chtimes(target, header.AccessTime, header.ModTime); err != nil {
					return err
				}
			}
		}
	}

	return unpack(tarPath, destPath, decompressor)
}

// unzip extracts a zip folder to a temporary directory,
// renaming it to the desired location
func unzip(ctx context.Context, zipPath string, destPath string) error {
	decompressor := func(src string, dest string) error {
		r, err := zip.OpenReader(src)
		if err != nil {
			return err
		}
		defer func() {
			if err := r.Close(); err != nil {
				panic(err)
			}
		}()

		// Closure to address file descriptors issue with all the deferred .Close() methods
		extractAndWriteFile := func(f *zip.File) error {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() {
				if err := rc.Close(); err != nil {
					panic(err)
				}
			}()

			path := filepath.Join(dest, f.Name)
			if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
				return fmt.Errorf("%s: Illegal file path", path)
			}

			if f.FileInfo().IsDir() {
				if err = os.MkdirAll(path, f.Mode()); err != nil {
					return err
				}
			} else {
				if err = os.MkdirAll(filepath.Dir(path), f.Mode()); err != nil {
					return err
				}
				f, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return err
				}
				defer func() {
					if err := f.Close(); err != nil {
						panic(err)
					}
				}()

				_, err = io.Copy(f, rc)
				if err != nil {
					return err
				}
			}
			return nil
		}

		logger := logging.RequireLoggerFromContext(ctx)
		for _, f := range r.File {
			if err := extractAndWriteFile(f); err != nil {
				return err
			}
			logger.WithFields(logging.Fields{"name": f.Name, "src": src}).Info(ctx, "Extracting file")
		}

		logger.WithField("src", src).Info(ctx, "Extraction finished")

		return nil
	}

	return unpack(zipPath, destPath, decompressor)
}

// unpack unpacks a compressed file (tarball or zip file) to a temporary directory,
// renaming it to the desired location
// decompression is done using the decompressor closure, that should decompress a tarball or zip file
func unpack(srcPath string, destPath string, decompressor func(string, string) error) error {
	// first extract the tar into a temporary dir
	tmpDir := destPath + ".tmpdir"
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}
	if decompressor != nil {
		if err = decompressor(srcPath, tmpDir); err != nil {
			return err
		}
	}
	// next, decide how we wish to rename the file/dir
	// to the destination path.
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}
	if len(files) == 1 {
		// if the tar is comprised of single file or directory,
		// rename that file to the desired location
		filePath := path.Join(tmpDir, files[0].Name())
		err = os.Rename(filePath, destPath)
		if err != nil {
			return argoerrs.InternalWrapError(err)
		}
		err = os.Remove(tmpDir)
		if err != nil {
			return argoerrs.InternalWrapError(err)
		}
	} else {
		// the tar extracted into multiple files. In this case,
		// just rename the temp directory to the dest path
		err = os.Rename(tmpDir, destPath)
		if err != nil {
			return argoerrs.InternalWrapError(err)
		}
	}
	return nil
}

func chmod(artPath string, mode int32, recurse bool) error {
	err := os.Chmod(artPath, os.FileMode(mode))
	if err != nil {
		return argoerrs.InternalWrapError(err)
	}

	if recurse {
		err = filepath.Walk(artPath, func(path string, f os.FileInfo, err error) error {
			return os.Chmod(path, os.FileMode(mode))
		})
		if err != nil {
			return argoerrs.InternalWrapError(err)
		}
	}

	return nil
}

// Wait is the sidecar container logic which waits for the main container to complete.
// Also monitors for updates in the pod annotations which may change (e.g. terminate)
// Upon completion, kills any sidecars after it finishes.
func (we *WorkflowExecutor) Wait(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	containerNames := we.Template.GetMainContainerNames()
	// only monitor progress if both tick durations are >0
	if we.annotationPatchTickDuration != 0 && we.readProgressFileTickDuration != 0 {
		go we.monitorProgress(ctx, os.Getenv(common.EnvVarProgressFile))
	} else {
		logger.WithField("annotationPatchTickDuration", we.annotationPatchTickDuration).WithField("readProgressFileTickDuration", we.readProgressFileTickDuration).Info(ctx, "monitoring progress disabled")
	}

	go we.monitorDeadline(ctx, containerNames)

	err := retryutil.OnError(executorretry.ExecutorRetry(ctx), func(err error) bool {
		return errorsutil.IsTransientErr(ctx, err)
	}, func() error {
		return we.RuntimeExecutor.Wait(ctx, containerNames)
	})

	logger.WithError(err).Info(ctx, "Main container completed")

	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("failed to wait for main container to complete: %w", err)
	}
	return nil
}

// monitorProgress monitors for self-reported progress in the progressFile and patches the pod annotations with the parsed progress.
//
// The function reads the last line of the `progressFile` every `readFileTickDuration`.
// If the line matches `N/M`, will set the progress annotation to the parsed progress value.
// Every `annotationPatchTickDuration` the pod is patched with the updated annotations. This way the controller
// gets notified of new self reported progress.
func (we *WorkflowExecutor) monitorProgress(ctx context.Context, progressFile string) {
	logger := logging.RequireLoggerFromContext(ctx)
	annotationPatchTicker := time.NewTicker(we.annotationPatchTickDuration)
	defer annotationPatchTicker.Stop()
	fileTicker := time.NewTicker(we.readProgressFileTickDuration)
	defer fileTicker.Stop()

	lastLine := ""
	progressFile = filepath.Clean(progressFile)

	for {
		select {
		case <-ctx.Done():
			logger.WithError(ctx.Err()).Info(ctx, "stopping progress monitor (context done)")
			return
		case <-annotationPatchTicker.C:
			if err := we.reportResult(ctx, wfv1.NodeResult{Progress: we.progress}); err != nil {
				logger.WithError(err).Info(ctx, "failed to report progress")
			} else {
				we.progress = ""
			}
		case <-fileTicker.C:
			data, err := os.ReadFile(progressFile)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					logger.WithError(err).WithField("file", progressFile).Info(ctx, "unable to watch file")
				}
				continue
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			mostRecent := strings.TrimSpace(lines[len(lines)-1])

			if mostRecent == "" || mostRecent == lastLine {
				continue
			}
			lastLine = mostRecent

			if progress, ok := wfv1.ParseProgress(lastLine); ok {
				logger.WithField("progress", progress).Info(ctx, "")
				we.progress = progress
			} else {
				logger.WithField("line", lastLine).Info(ctx, "unable to parse progress")
			}
		}
	}
}

// monitorDeadline checks to see if we exceeded the deadline for the step and
// terminates the main container if we did
func (we *WorkflowExecutor) monitorDeadline(ctx context.Context, containerNames []string) {
	deadlineExceeded := make(chan bool, 1)
	if !we.Deadline.IsZero() {
		t := time.AfterFunc(time.Until(we.Deadline), func() {
			deadlineExceeded <- true
		})
		defer t.Stop()
	}

	var message string
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("containers", containerNames).Info(ctx, "Starting deadline monitor")
	select {
	case <-ctx.Done():
		logger.Info(ctx, "Deadline monitor stopped")
		return
	case <-deadlineExceeded:
		message = "Step exceeded its deadline"
	}
	logger.Info(ctx, message)
	util.WriteTerminateMessage(message)

	containerNames = slices.DeleteFunc(containerNames, common.IsArtifactPluginSidecar)
	we.killContainers(ctx, containerNames)
}

func (we *WorkflowExecutor) killContainers(ctx context.Context, containerNames []string) {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("containerNames", containerNames).Info(ctx, "Killing containers")
	terminationGracePeriodDuration := GetTerminationGracePeriodDuration()
	if err := we.RuntimeExecutor.Kill(ctx, containerNames, terminationGracePeriodDuration); err != nil {
		logger.WithField("containerNames", containerNames).WithError(err).Warn(ctx, "Failed to kill")
	}
}

func (we *WorkflowExecutor) Init() error {
	if i, ok := we.RuntimeExecutor.(Initializer); ok {
		return i.Init(we.Template)
	}
	return nil
}

func (we *WorkflowExecutor) KillArtifactSidecars(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	pluginNamesEnv := os.Getenv(common.EnvVarArtifactPluginNames)
	if pluginNamesEnv == "" {
		logger.Info(ctx, "no artifact sidecars to kill")
		return nil
	}
	artifactSidecars := strings.Split(pluginNamesEnv, ",")
	logger.WithFields(logging.Fields{"numSidecars": len(artifactSidecars), "artifactSidecars": artifactSidecars}).Info(ctx, "killing artifact sidecars")
	err := we.RuntimeExecutor.Kill(ctx, artifactSidecars, GetTerminationGracePeriodDuration())
	if err != nil {
		logger.WithError(err).WithFields(logging.Fields{"artifactSidecars": artifactSidecars}).Error(ctx, "failed to kill artifact sidecars")
		return err
	}
	logger.WithFields(logging.Fields{"artifactSidecars": artifactSidecars}).Info(ctx, "artifact sidecars killed")
	return nil
}
