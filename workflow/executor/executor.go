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
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/file"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	retryutil "k8s.io/client-go/util/retry"

	argoerrs "github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoprojv1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/archive"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	artifact "github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	artifactcommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	executorretry "github.com/argoproj/argo-workflows/v3/workflow/executor/retry"
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
}

type Initializer interface {
	Init(tmpl wfv1.Template) error
}

//go:generate mockery --name=ContainerRuntimeExecutor

// ContainerRuntimeExecutor is the interface for interacting with a container runtime
type ContainerRuntimeExecutor interface {
	// GetFileContents returns the file contents of a file in a container as a string
	GetFileContents(containerName string, sourcePath string) (string, error)

	// CopyFile copies a source file in a container to a local path
	CopyFile(containerName, sourcePath, destPath string, compressionLevel int) error

	// GetOutputStream returns the entirety of the container output as a io.Reader
	// Used to capture script results as an output parameter, and to archive container logs
	GetOutputStream(ctx context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error)

	// Wait waits for the container to complete.
	Wait(ctx context.Context, containerNames []string) error

	// Kill a list of containers first with a SIGTERM then with a SIGKILL after a grace period
	Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error
}

// NewExecutor instantiates a new workflow executor
func NewExecutor(
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
) WorkflowExecutor {
	log.WithFields(log.Fields{"Steps": executorretry.Steps, "Duration": executorretry.Duration, "Factor": executorretry.Factor, "Jitter": executorretry.Jitter}).Info("Using executor retry strategy")
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
		memoizedConfigMaps:           map[string]string{},
		memoizedSecrets:              map[string][]byte{},
		errors:                       []error{},
		annotationPatchTickDuration:  annotationPatchTickDuration,
		readProgressFileTickDuration: readProgressFileTickDuration,
	}
}

// HandleError is a helper to annotate the pod with the error message upon a unexpected executor panic or error
func (we *WorkflowExecutor) HandleError(ctx context.Context) {
	if r := recover(); r != nil {
		util.WriteTerminateMessage(fmt.Sprintf("%v", r))
		log.Fatalf("executor panic: %+v\n%s", r, debug.Stack())
	} else {
		if len(we.errors) > 0 {
			util.WriteTerminateMessage(we.errors[0].Error())
		}
	}
}

// LoadArtifacts loads artifacts from location to a container path
func (we *WorkflowExecutor) LoadArtifacts(ctx context.Context) error {
	log.Infof("Start loading input artifacts...")
	for _, art := range we.Template.Inputs.Artifacts {

		log.Infof("Downloading artifact: %s", art.Name)

		if !art.HasLocationOrKey() {
			if art.Optional {
				log.Warnf("Ignoring optional artifact '%s' which was not supplied", art.Name)
				continue
			} else {
				return argoerrs.Errorf(argoerrs.CodeNotFound, "required artifact '%s' not supplied", art.Name)
			}
		}
		err := art.CleanPath()
		if err != nil {
			return err
		}
		driverArt, err := we.newDriverArt(&art)
		if err != nil {
			return fmt.Errorf("failed to load artifact '%s': %w", art.Name, err)
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
			log.Infof("Specified artifact path %s overlaps with volume mount at %s. Extracting to volume mount", art.Path, mnt.MountPath)
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
		err = artDriver.Load(driverArt, tempArtPath)
		if err != nil {
			if art.Optional && argoerrs.IsCode(argoerrs.CodeNotFound, err) {
				log.Infof("Skipping optional input artifact that was not found: %s", art.Name)
				continue
			}
			return fmt.Errorf("artifact %s failed to load: %w", art.Name, err)
		}

		isTar := false
		isZip := false
		if art.GetArchive().None != nil {
			// explicitly not a tar
			isTar = false
			isZip = false
		} else if art.GetArchive().Tar != nil {
			// explicitly a tar
			isTar = true
		} else if art.GetArchive().Zip != nil {
			// explicitly a zip
			isZip = true
		} else {
			// auto-detect if tarball
			// (don't try to autodetect zip files for backwards compatibility)
			isTar, err = isTarball(tempArtPath)
			if err != nil {
				return err
			}
		}

		if isTar {
			err = untar(tempArtPath, artPath)
			_ = os.Remove(tempArtPath)
		} else if isZip {
			err = unzip(tempArtPath, artPath)
			_ = os.Remove(tempArtPath)
		} else {
			err = os.Rename(tempArtPath, artPath)
		}
		if err != nil {
			return err
		}

		log.Infof("Successfully download file: %s", artPath)
		if art.Mode != nil {
			err = chmod(artPath, *art.Mode, art.RecurseMode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// StageFiles will create any files required by script/resource templates
func (we *WorkflowExecutor) StageFiles() error {
	var filePath string
	var body []byte
	mode := os.FileMode(0o644)
	switch we.Template.GetType() {
	case wfv1.TemplateTypeScript:
		log.Infof("Loading script source to %s", common.ExecutorScriptSourcePath)
		filePath = common.ExecutorScriptSourcePath
		body = []byte(we.Template.Script.Source)
		mode = os.FileMode(0o755)
	case wfv1.TemplateTypeResource:
		if we.Template.Resource.ManifestFrom != nil && we.Template.Resource.ManifestFrom.Artifact != nil {
			log.Infof("manifest %s already staged", we.Template.Resource.ManifestFrom.Artifact.Name)
			return nil
		}
		log.Infof("Loading manifest to %s", common.ExecutorResourceManifestPath)
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
	artifacts := wfv1.Artifacts{}
	if len(we.Template.Outputs.Artifacts) == 0 {
		log.Infof("No output artifacts")
		return artifacts, nil
	}

	log.Infof("Saving output artifacts")
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
	} else {
		return artifacts, errors.New(aggregateError)
	}

}

// save artifact
// return whether artifact was in fact saved, and if there was an error
func (we *WorkflowExecutor) saveArtifact(ctx context.Context, containerName string, art *wfv1.Artifact) (bool, error) {
	// Determine the file path of where to find the artifact
	err := art.CleanPath()
	if err != nil {
		return false, err
	}
	fileName, localArtPath, err := we.stageArchiveFile(containerName, art)
	if err != nil {
		if art.Optional && argoerrs.IsCode(argoerrs.CodeNotFound, err) {
			log.Warnf("Ignoring optional artifact '%s' which does not exist in path '%s': %v", art.Name, art.Path, err)
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
		log.Warnf("The file %q is empty. It may not be uploaded successfully depending on the artifact driver", localArtPath)
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
	err = artDriver.Save(localArtPath, driverArt)
	if err != nil {
		return err
	}
	we.maybeDeleteLocalArtPath(localArtPath)
	log.Infof("Successfully saved file: %s", localArtPath)
	return nil
}

func (we *WorkflowExecutor) maybeDeleteLocalArtPath(localArtPath string) {
	if os.Getenv("REMOVE_LOCAL_ART_PATH") == "true" {
		log.WithField("localArtPath", localArtPath).Info("deleting local artifact")
		// remove is best effort (the container will go away anyways).
		// we just want reduce peak space usage
		err := os.Remove(localArtPath)
		if err != nil {
			log.Warnf("Failed to remove %s: %v", localArtPath, err)
		}
	} else {
		log.WithField("localArtPath", localArtPath).Info("not deleting local artifact")
	}
}

// stageArchiveFile stages a path in a container for archiving from the wait sidecar.
// Returns a filename and a local path for the upload.
// The filename is incorporated into the final path when uploading it to the artifact repo.
// The local path is the final staging location of the file (or directory) which we will pass
// to the SaveArtifacts call and may be a directory or file.
func (we *WorkflowExecutor) stageArchiveFile(containerName string, art *wfv1.Artifact) (string, string, error) {
	log.Infof("Staging artifact: %s", art.Name)
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
		log.Infof("Staging %s from mirrored volume mount %s", art.Path, mountedArtPath)
		if strategy.None != nil {
			fileName := filepath.Base(art.Path)
			log.Infof("No compression strategy needed. Staging skipped")
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
			err = archive.ZipToWriter(mountedArtPath, zw)
			if err != nil {
				return "", "", err
			}
			log.Infof("Successfully staged %s from mirrored volume mount %s", art.Path, mountedArtPath)
			return fileName, localArtPath, nil
		}
		fileName := fmt.Sprintf("%s.tgz", art.Name)
		localArtPath := filepath.Join(tempOutArtDir, fileName)
		f, err := os.Create(localArtPath)
		if err != nil {
			return "", "", argoerrs.InternalWrapError(err)
		}
		w := bufio.NewWriter(f)
		err = archive.TarGzToWriter(mountedArtPath, compressionLevel, w)
		if err != nil {
			return "", "", err
		}
		log.Infof("Successfully staged %s from mirrored volume mount %s", art.Path, mountedArtPath)
		return fileName, localArtPath, nil
	}

	fileName := fmt.Sprintf("%s.tgz", art.Name)
	localArtPath := filepath.Join(tempOutArtDir, fileName)
	log.Infof("Copying %s from container base image layer to %s", art.Path, localArtPath)

	err := we.RuntimeExecutor.CopyFile(containerName, art.Path, localArtPath, compressionLevel)
	if err != nil {
		return "", "", err
	}
	if strategy.Tar != nil {
		// NOTE we already tar gzip the file in the executor. So this is a noop.
		return fileName, localArtPath, nil
	}
	// localArtPath now points to a .tgz file, and the archive strategy is *not* tar. We need to untar it
	log.Infof("Untaring %s archive before upload", localArtPath)
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
		err = archive.ZipToWriter(unarchivedArtPath, zw)
		if err != nil {
			return "", "", err
		}
		log.Infof("Successfully zipped %s to %s", unarchivedArtPath, localArtPath)
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
	if len(we.Template.Outputs.Parameters) == 0 {
		log.Infof("No output parameters")
		return nil
	}
	log.Infof("Saving output parameters")
	for i, param := range we.Template.Outputs.Parameters {
		log.Infof("Saving path output parameter: %s", param.Name)
		// Determine the file path of where to find the parameter
		if param.ValueFrom == nil || param.ValueFrom.Path == "" {
			continue
		}

		var output *wfv1.AnyString
		if we.isBaseImagePath(param.ValueFrom.Path) {
			log.Infof("Copying %s from base image layer", param.ValueFrom.Path)
			fileContents, err := we.RuntimeExecutor.GetFileContents(common.MainContainerName, param.ValueFrom.Path)
			if err != nil {
				// We have a default value to use instead of returning an error
				if param.ValueFrom.Default != nil {
					output = param.ValueFrom.Default
				} else {
					return err
				}
			} else {
				output = wfv1.AnyStringPtr(fileContents)
			}
		} else {
			log.Infof("Copying %s from volume mount", param.ValueFrom.Path)
			mountedPath := filepath.Join(common.ExecutorMainFilesystemDir, param.ValueFrom.Path)
			data, err := os.ReadFile(filepath.Clean(mountedPath))
			if err != nil {
				// We have a default value to use instead of returning an error
				if param.ValueFrom.Default != nil {
					output = param.ValueFrom.Default
				} else {
					return err
				}
			} else {
				output = wfv1.AnyStringPtr(string(data))
			}
		}

		// Trims off a single newline for user convenience
		output = wfv1.AnyStringPtr(strings.TrimSuffix(output.String(), "\n"))
		we.Template.Outputs.Parameters[i].Value = output
		log.Infof("Successfully saved output parameter: %s", param.Name)
	}
	return nil
}

func (we *WorkflowExecutor) SaveLogs(ctx context.Context) []wfv1.Artifact {
	var logArtifacts []wfv1.Artifact
	tempLogsDir := "/tmp/argo/outputs/logs"

	if we.Template.SaveLogsAsArtifact() {
		err := os.MkdirAll(tempLogsDir, os.ModePerm)
		if err != nil {
			we.AddError(argoerrs.InternalWrapError(err))
		}

		containerNames := we.Template.GetMainContainerNames()
		logArtifacts = make([]wfv1.Artifact, 0)

		for _, containerName := range containerNames {
			// Saving logs
			art, err := we.saveContainerLogs(ctx, tempLogsDir, containerName)
			if err != nil {
				we.AddError(err)
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
	driver, err := artifact.NewDriver(ctx, art, we)
	if err == artifact.ErrUnsupportedDriver {
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
	err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		configmap, err = configmapsIf.Get(ctx, name, metav1.GetOptions{})
		return !errorsutil.IsTransientErr(err), err
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

// GetSecrets retrieves a secret value and memoizes the result
func (we *WorkflowExecutor) GetSecrets(ctx context.Context, namespace, name, key string) ([]byte, error) {
	cachedKey := fmt.Sprintf("%s/%s/%s", namespace, name, key)
	if val, ok := we.memoizedSecrets[cachedKey]; ok {
		return val, nil
	}
	secretsIf := we.ClientSet.CoreV1().Secrets(namespace)
	var secret *apiv1.Secret
	err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		secret, err = secretsIf.Get(ctx, name, metav1.GetOptions{})
		return !errorsutil.IsTransientErr(err), err
	})
	if err != nil {
		return []byte{}, argoerrs.InternalWrapError(err)
	}
	// memoize all keys in the secret since it's highly likely we will need to get a
	// subsequent key in the secret (e.g. username + password) and we can save an API call
	for k, v := range secret.Data {
		we.memoizedSecrets[fmt.Sprintf("%s/%s/%s", namespace, name, k)] = v
	}
	val, ok := we.memoizedSecrets[cachedKey]
	if !ok {
		return []byte{}, argoerrs.Errorf(argoerrs.CodeBadRequest, "secret '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// GetTerminationGracePeriodDuration returns the terminationGracePeriodSeconds of podSpec in Time.Duration format
func getTerminationGracePeriodDuration() time.Duration {
	x, _ := strconv.ParseInt(os.Getenv(common.EnvVarTerminationGracePeriodSeconds), 10, 64)
	if x > 0 {
		return time.Duration(x) * time.Second
	}
	return 30 * time.Second
}

// CaptureScriptResult will add the stdout of a script template as output result
func (we *WorkflowExecutor) CaptureScriptResult(ctx context.Context) error {
	if !we.IncludeScriptOutput {
		log.Infof("No Script output reference in workflow. Capturing script output ignored")
		return nil
	}
	if !we.Template.HasOutput() {
		log.Infof("Template type is neither of Script, Container, or Pod. Capturing script output ignored")
		return nil
	}
	log.Infof("Capturing script output")
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
		log.Warnf("Output is larger than the maximum allowed size of 256 kB, only the last 256 kB were saved")
		out = out[len(out)-maxAnnotationSize:]
	}

	we.Template.Outputs.Result = &out
	return nil
}

// FinalizeOutput adds a label or annotation to denote that outputs have completed reporting.
func (we *WorkflowExecutor) FinalizeOutput(ctx context.Context) {
	err := retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    5,
		Cap:      30 * time.Second,
	}, errorsutil.IsTransientErr, func() error {
		err := we.patchTaskResultLabels(ctx, map[string]string{
			common.LabelKeyReportOutputsCompleted: "true",
		})
		if apierr.IsForbidden(err) || apierr.IsNotFound(err) {
			log.WithError(err).Warn("failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		}
		return err
	})
	if err != nil {
		we.AddError(err)
	}
}

func (we *WorkflowExecutor) InitializeOutput(ctx context.Context) {
	err := retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    5,
		Cap:      30 * time.Second,
	}, errorsutil.IsTransientErr, func() error {
		err := we.upsertTaskResult(ctx, wfv1.NodeResult{})
		if apierr.IsForbidden(err) {
			log.WithError(err).Warn("failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		}
		return err
	})
	if err != nil {
		we.AddError(err)
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

func shouldLogRetry(attempt uint64) bool {
	switch {
	case attempt < 10:
		return true // log first 10 attempts
	case attempt < 100:
		return attempt%10 == 0 // log every 10 attempts (10–99)
	case attempt < 1000:
		return attempt%50 == 0 // log every 50 attempts (100–999)
	default:
		return attempt%100 == 0 // log every 100 attempts (1000+)
	}
}

func (we *WorkflowExecutor) reportResult(ctx context.Context, result wfv1.NodeResult) error {
	var count uint64 // used to avoid spamming with these messages
	return retryutil.OnError(wait.Backoff{
		Duration: time.Second,
		Factor:   2.0,
		Jitter:   0.2,
		Steps:    math.MaxInt32, // effectively infinite retries
		Cap:      30 * time.Second,
	}, errorsutil.IsTransientErr, func() error {
		err := we.upsertTaskResult(ctx, result)
		if apierr.IsForbidden(err) && shouldLogRetry(count) {
			log.WithError(err).Warn("failed to patch task result, see https://argo-workflows.readthedocs.io/en/latest/workflow-rbac/")
		}
		count++
		return err
	})
}

// AddError adds an error to the list of encountered errors during execution
func (we *WorkflowExecutor) AddError(err error) {
	log.Errorf("executor error: %+v", err)
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
	data, err := json.Marshal(map[string]interface{}{"metadata": metav1.ObjectMeta{
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
func isTarball(filePath string) (bool, error) {
	log.Infof("Detecting if %s is a tarball", filePath)
	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("Error closing file[%s]: %v", filePath, err)
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
			case err == io.EOF:
				return nil
			case err != nil:
				return err
			case header == nil:
				continue
			}
			target := filepath.Join(dest, filepath.Clean(header.Name))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil && os.IsExist(err) {
				return err
			}
			switch header.Typeflag {
			case tar.TypeSymlink:
				err := os.Symlink(header.Linkname, target)
				if err != nil {
					return err
				}
			case tar.TypeDir:
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
			case tar.TypeReg:
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
func unzip(zipPath string, destPath string) error {
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

			path := filepath.Join(dest, f.Name) //nolint:gosec
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

				_, err = io.Copy(f, rc) //nolint:gosec
				if err != nil {
					return err
				}
			}
			return nil
		}

		for _, f := range r.File {
			if err := extractAndWriteFile(f); err != nil {
				return err
			}
			log.Infof("Extracting file: %s", f.Name)
		}

		log.Infof("Extraction of %s finished!", src)

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
	containerNames := we.Template.GetMainContainerNames()
	// only monitor progress if both tick durations are >0
	if we.annotationPatchTickDuration != 0 && we.readProgressFileTickDuration != 0 {
		go we.monitorProgress(ctx, os.Getenv(common.EnvVarProgressFile))
	} else {
		log.WithField("annotationPatchTickDuration", we.annotationPatchTickDuration).WithField("readProgressFileTickDuration", we.readProgressFileTickDuration).Info("monitoring progress disabled")
	}

	go we.monitorDeadline(ctx, containerNames)

	err := retryutil.OnError(executorretry.ExecutorRetry, errorsutil.IsTransientErr, func() error {
		return we.RuntimeExecutor.Wait(ctx, containerNames)
	})

	log.WithError(err).Info("Main container completed")

	if err != nil && err != context.Canceled {
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
	annotationPatchTicker := time.NewTicker(we.annotationPatchTickDuration)
	defer annotationPatchTicker.Stop()
	fileTicker := time.NewTicker(we.readProgressFileTickDuration)
	defer fileTicker.Stop()

	lastLine := ""
	progressFile = filepath.Clean(progressFile)

	for {
		select {
		case <-ctx.Done():
			log.WithError(ctx.Err()).Info("stopping progress monitor (context done)")
			return
		case <-annotationPatchTicker.C:
			if err := we.reportResult(ctx, wfv1.NodeResult{Progress: we.progress}); err != nil {
				log.WithError(err).Info("failed to report progress")
			} else {
				we.progress = ""
			}
		case <-fileTicker.C:
			data, err := os.ReadFile(progressFile)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					log.WithError(err).WithField("file", progressFile).Info("unable to watch file")
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
				log.WithField("progress", progress).Info()
				we.progress = progress
			} else {
				log.WithField("line", lastLine).Info("unable to parse progress")
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
	log.Infof("Starting deadline monitor")
	select {
	case <-ctx.Done():
		log.Info("Deadline monitor stopped")
		return
	case <-deadlineExceeded:
		message = "Step exceeded its deadline"
	}
	log.Info(message)
	util.WriteTerminateMessage(message)
	we.killContainers(ctx, containerNames)
}

func (we *WorkflowExecutor) killContainers(ctx context.Context, containerNames []string) {
	log.Infof("Killing containers")
	terminationGracePeriodDuration := getTerminationGracePeriodDuration()
	if err := we.RuntimeExecutor.Kill(ctx, containerNames, terminationGracePeriodDuration); err != nil {
		log.Warnf("Failed to kill %q: %v", containerNames, err)
	}
}

func (we *WorkflowExecutor) Init() error {
	if i, ok := we.RuntimeExecutor.(Initializer); ok {
		return i.Init(we.Template)
	}
	return nil
}
