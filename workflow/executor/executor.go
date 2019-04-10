package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	argofile "github.com/argoproj/pkg/file"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/archive"
	"github.com/argoproj/argo/util/retry"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/artifacts/artifactory"
	"github.com/argoproj/argo/workflow/artifacts/git"
	"github.com/argoproj/argo/workflow/artifacts/hdfs"
	"github.com/argoproj/argo/workflow/artifacts/http"
	"github.com/argoproj/argo/workflow/artifacts/raw"
	"github.com/argoproj/argo/workflow/artifacts/s3"
	"github.com/argoproj/argo/workflow/common"
)

const (
	// This directory temporarily stores the tarballs of the artifacts before uploading
	tempOutArtDir = "/argo/outputs/artifacts"
)

// WorkflowExecutor is program which runs as the init/wait container
type WorkflowExecutor struct {
	common.ResourceInterface

	PodName            string
	Template           wfv1.Template
	ClientSet          kubernetes.Interface
	Namespace          string
	PodAnnotationsPath string
	ExecutionControl   *common.ExecutionControl
	RuntimeExecutor    ContainerRuntimeExecutor

	// memoized container ID to prevent multiple lookups
	mainContainerID string
	// memoized configmaps
	memoizedConfigMaps map[string]string
	// memoized secrets
	memoizedSecrets map[string][]byte
	// list of errors that occurred during execution.
	// the first of these is used as the overall message of the node
	errors []error
}

// ContainerRuntimeExecutor is the interface for interacting with a container runtime (e.g. docker)
type ContainerRuntimeExecutor interface {
	// GetFileContents returns the file contents of a file in a container as a string
	GetFileContents(containerID string, sourcePath string) (string, error)

	// CopyFile copies a source file in a container to a local path
	CopyFile(containerID string, sourcePath string, destPath string) error

	// GetOutputStream returns the entirety of the container output as a io.Reader
	// Used to capture script results as an output parameter, and to archive container logs
	GetOutputStream(containerID string, combinedOutput bool) (io.ReadCloser, error)

	// WaitInit is called before Wait() to signal the executor about an impending Wait call.
	// For most executors this is a noop, and is only used by the the PNS executor
	WaitInit() error

	// Wait waits for the container to complete
	Wait(containerID string) error

	// Kill a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
	Kill(containerIDs []string) error
}

// NewExecutor instantiates a new workflow executor
func NewExecutor(clientset kubernetes.Interface, podName, namespace, podAnnotationsPath string, cre ContainerRuntimeExecutor, template wfv1.Template) WorkflowExecutor {
	return WorkflowExecutor{
		PodName:            podName,
		ClientSet:          clientset,
		Namespace:          namespace,
		PodAnnotationsPath: podAnnotationsPath,
		RuntimeExecutor:    cre,
		Template:           template,
		memoizedConfigMaps: map[string]string{},
		memoizedSecrets:    map[string][]byte{},
		errors:             []error{},
	}
}

// HandleError is a helper to annotate the pod with the error message upon a unexpected executor panic or error
func (we *WorkflowExecutor) HandleError() {
	if r := recover(); r != nil {
		_ = we.AddAnnotation(common.AnnotationKeyNodeMessage, fmt.Sprintf("%v", r))
		log.Fatalf("executor panic: %+v\n%s", r, debug.Stack())
	} else {
		if len(we.errors) > 0 {
			_ = we.AddAnnotation(common.AnnotationKeyNodeMessage, we.errors[0].Error())
		}
	}
}

// LoadArtifacts loads artifacts from location to a container path
func (we *WorkflowExecutor) LoadArtifacts() error {
	log.Infof("Start loading input artifacts...")

	for _, art := range we.Template.Inputs.Artifacts {

		log.Infof("Downloading artifact: %s", art.Name)

		if !art.HasLocation() {
			if art.Optional {
				log.Warnf("Ignoring optional artifact '%s' which was not supplied", art.Name)
				continue
			} else {
				return errors.New("required artifact %s not supplied", art.Name)
			}
		}
		artDriver, err := we.InitDriver(art)
		if err != nil {
			return err
		}
		// Determine the file path of where to load the artifact
		if art.Path == "" {
			return errors.InternalErrorf("Artifact %s did not specify a path", art.Name)
		}
		var artPath string
		mnt := common.FindOverlappingVolume(&we.Template, art.Path)
		if mnt == nil {
			artPath = path.Join(common.ExecutorArtifactBaseDir, art.Name)
		} else {
			// If we get here, it means the input artifact path overlaps with an user specified
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
		err = artDriver.Load(&art, tempArtPath)
		if err != nil {
			return err
		}
		if isTarball(tempArtPath) {
			err = untar(tempArtPath, artPath)
			_ = os.Remove(tempArtPath)
		} else {
			err = os.Rename(tempArtPath, artPath)
		}
		if err != nil {
			return err
		}

		log.Infof("Successfully download file: %s", artPath)
		if art.Mode != nil {
			err = os.Chmod(artPath, os.FileMode(*art.Mode))
			if err != nil {
				return errors.InternalWrapError(err)
			}
		}
	}
	return nil
}

// StageFiles will create any files required by script/resource templates
func (we *WorkflowExecutor) StageFiles() error {
	var filePath string
	var body []byte
	switch we.Template.GetType() {
	case wfv1.TemplateTypeScript:
		log.Infof("Loading script source to %s", common.ExecutorScriptSourcePath)
		filePath = common.ExecutorScriptSourcePath
		body = []byte(we.Template.Script.Source)
	case wfv1.TemplateTypeResource:
		log.Infof("Loading manifest to %s", common.ExecutorResourceManifestPath)
		filePath = common.ExecutorResourceManifestPath
		body = []byte(we.Template.Resource.Manifest)
	default:
		return nil
	}
	err := ioutil.WriteFile(filePath, body, 0644)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

// SaveArtifacts uploads artifacts to the archive location
func (we *WorkflowExecutor) SaveArtifacts() error {
	if len(we.Template.Outputs.Artifacts) == 0 {
		log.Infof("No output artifacts")
		return nil
	}
	log.Infof("Saving output artifacts")
	mainCtrID, err := we.GetMainContainerID()
	if err != nil {
		return err
	}

	err = os.MkdirAll(tempOutArtDir, os.ModePerm)
	if err != nil {
		return errors.InternalWrapError(err)
	}

	for i, art := range we.Template.Outputs.Artifacts {
		err := we.saveArtifact(mainCtrID, &art)
		if err != nil {
			return err
		}
		we.Template.Outputs.Artifacts[i] = art
	}
	return nil
}

func (we *WorkflowExecutor) saveArtifact(mainCtrID string, art *wfv1.Artifact) error {
	// Determine the file path of where to find the artifact
	if art.Path == "" {
		return errors.InternalErrorf("Artifact %s did not specify a path", art.Name)
	}
	fileName, localArtPath, err := we.stageArchiveFile(mainCtrID, art)
	if err != nil {
		if art.Optional && errors.IsCode(errors.CodeNotFound, err) {
			log.Warnf("Ignoring optional artifact '%s' which does not exist in path '%s': %v", art.Name, art.Path, err)
			return nil
		}
		return err
	}
	if !art.HasLocation() {
		// If user did not explicitly set an artifact destination location in the template,
		// use the default archive location (appended with the filename).
		if we.Template.ArchiveLocation == nil {
			return errors.Errorf(errors.CodeBadRequest, "Unable to determine path to store %s. No archive location", art.Name)
		}
		if we.Template.ArchiveLocation.S3 != nil {
			shallowCopy := *we.Template.ArchiveLocation.S3
			art.S3 = &shallowCopy
			art.S3.Key = path.Join(art.S3.Key, fileName)
		} else if we.Template.ArchiveLocation.Artifactory != nil {
			shallowCopy := *we.Template.ArchiveLocation.Artifactory
			art.Artifactory = &shallowCopy
			artifactoryURL, urlParseErr := url.Parse(art.Artifactory.URL)
			if urlParseErr != nil {
				return urlParseErr
			}
			artifactoryURL.Path = path.Join(artifactoryURL.Path, fileName)
			art.Artifactory.URL = artifactoryURL.String()
		} else if we.Template.ArchiveLocation.HDFS != nil {
			shallowCopy := *we.Template.ArchiveLocation.HDFS
			art.HDFS = &shallowCopy
			art.HDFS.Path = path.Join(art.HDFS.Path, fileName)
		} else {
			return errors.Errorf(errors.CodeBadRequest, "Unable to determine path to store %s. Archive location provided no information", art.Name)
		}
	}

	artDriver, err := we.InitDriver(*art)
	if err != nil {
		return err
	}
	err = artDriver.Save(localArtPath, art)
	if err != nil {
		return err
	}
	// remove is best effort (the container will go away anyways).
	// we just want reduce peak space usage
	err = os.Remove(localArtPath)
	if err != nil {
		log.Warnf("Failed to remove %s: %v", localArtPath, err)
	}
	log.Infof("Successfully saved file: %s", localArtPath)
	return nil
}

// stageArchiveFile stages a path in a container for archiving from the wait sidecar.
// Returns a filename and a local path for the upload.
// The filename is incorporated into the final path when uploading it to the artifact repo.
// The local path is the final staging location of the file (or directory) which we will pass
// to the SaveArtifacts call and may be a directory or file.
func (we *WorkflowExecutor) stageArchiveFile(mainCtrID string, art *wfv1.Artifact) (string, string, error) {
	log.Infof("Staging artifact: %s", art.Name)
	strategy := art.Archive
	if strategy == nil {
		// If no strategy is specified, default to the tar strategy
		strategy = &wfv1.ArchiveStrategy{
			Tar: &wfv1.TarStrategy{},
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
			return fileName, mountedArtPath, nil
		}
		fileName := fmt.Sprintf("%s.tgz", art.Name)
		localArtPath := filepath.Join(tempOutArtDir, fileName)
		f, err := os.Create(localArtPath)
		if err != nil {
			return "", "", errors.InternalWrapError(err)
		}
		w := bufio.NewWriter(f)
		err = archive.TarGzToWriter(mountedArtPath, w)
		if err != nil {
			return "", "", err
		}
		log.Infof("Successfully staged %s from mirrored volume mount %s", art.Path, mountedArtPath)
		return fileName, localArtPath, nil
	}

	fileName := fmt.Sprintf("%s.tgz", art.Name)
	localArtPath := filepath.Join(tempOutArtDir, fileName)
	log.Infof("Copying %s from container base image layer to %s", art.Path, localArtPath)

	err := we.RuntimeExecutor.CopyFile(mainCtrID, art.Path, localArtPath)
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
		return "", "", errors.InternalWrapError(err)
	}
	isDir, err := argofile.IsDirectory(unarchivedArtPath)
	if err != nil {
		return "", "", errors.InternalWrapError(err)
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
			return "", "", errors.InternalWrapError(err)
		}
	}
	// In the future, if we were to support other compression formats (e.g. bzip2) or options
	// the logic would go here, and compression would be moved out of the executors
	return fileName, localArtPath, nil
}

func (we *WorkflowExecutor) isBaseImagePath(path string) bool {
	return common.FindOverlappingVolume(&we.Template, path) == nil
}

// SaveParameters will save the content in the specified file path as output parameter value
func (we *WorkflowExecutor) SaveParameters() error {
	if len(we.Template.Outputs.Parameters) == 0 {
		log.Infof("No output parameters")
		return nil
	}
	log.Infof("Saving output parameters")
	mainCtrID, err := we.GetMainContainerID()
	if err != nil {
		return err
	}

	for i, param := range we.Template.Outputs.Parameters {
		log.Infof("Saving path output parameter: %s", param.Name)
		// Determine the file path of where to find the parameter
		if param.ValueFrom == nil || param.ValueFrom.Path == "" {
			continue
		}

		var output string
		if we.isBaseImagePath(param.ValueFrom.Path) {
			log.Infof("Copying %s from base image layer", param.ValueFrom.Path)
			output, err = we.RuntimeExecutor.GetFileContents(mainCtrID, param.ValueFrom.Path)
			if err != nil {
				return err
			}
		} else {
			log.Infof("Copying %s from from volume mount", param.ValueFrom.Path)
			mountedPath := filepath.Join(common.ExecutorMainFilesystemDir, param.ValueFrom.Path)
			out, err := ioutil.ReadFile(mountedPath)
			if err != nil {
				return err
			}
			output = string(out)
		}

		outputLen := len(output)
		// Trims off a single newline for user convenience
		if outputLen > 0 && output[outputLen-1] == '\n' {
			output = output[0 : outputLen-1]
		}
		we.Template.Outputs.Parameters[i].Value = &output
		log.Infof("Successfully saved output parameter: %s", param.Name)
	}
	return nil
}

// SaveLogs saves logs
func (we *WorkflowExecutor) SaveLogs() (*wfv1.Artifact, error) {
	if we.Template.ArchiveLocation == nil || we.Template.ArchiveLocation.ArchiveLogs == nil || !*we.Template.ArchiveLocation.ArchiveLogs {
		return nil, nil
	}
	log.Infof("Saving logs")
	mainCtrID, err := we.GetMainContainerID()
	if err != nil {
		return nil, err
	}
	tempLogsDir := "/argo/outputs/logs"
	err = os.MkdirAll(tempLogsDir, os.ModePerm)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fileName := "main.log"
	mainLog := path.Join(tempLogsDir, fileName)
	err = we.saveLogToFile(mainCtrID, mainLog)
	if err != nil {
		return nil, err
	}
	art := wfv1.Artifact{
		Name:             "main-logs",
		ArtifactLocation: *we.Template.ArchiveLocation,
	}
	if we.Template.ArchiveLocation.S3 != nil {
		shallowCopy := *we.Template.ArchiveLocation.S3
		art.S3 = &shallowCopy
		art.S3.Key = path.Join(art.S3.Key, fileName)
	} else if we.Template.ArchiveLocation.Artifactory != nil {
		shallowCopy := *we.Template.ArchiveLocation.Artifactory
		art.Artifactory = &shallowCopy
		artifactoryURL, urlParseErr := url.Parse(art.Artifactory.URL)
		if urlParseErr != nil {
			return nil, urlParseErr
		}
		artifactoryURL.Path = path.Join(artifactoryURL.Path, fileName)
		art.Artifactory.URL = artifactoryURL.String()
	} else if we.Template.ArchiveLocation.HDFS != nil {
		shallowCopy := *we.Template.ArchiveLocation.HDFS
		art.HDFS = &shallowCopy
		art.HDFS.Path = path.Join(art.HDFS.Path, fileName)
	} else {
		return nil, errors.Errorf(errors.CodeBadRequest, "Unable to determine path to store %s. Archive location provided no information", art.Name)
	}
	artDriver, err := we.InitDriver(art)
	if err != nil {
		return nil, err
	}
	err = artDriver.Save(mainLog, &art)
	if err != nil {
		return nil, err
	}

	return &art, nil
}

// GetSecretFromVolMount will retrive the Secrets from VolumeMount
func (we *WorkflowExecutor) GetSecretFromVolMount(accessKeyName string, accessKey string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(common.SecretVolMountPath, accessKeyName, accessKey))
}

// saveLogToFile saves the entire log output of a container to a local file
func (we *WorkflowExecutor) saveLogToFile(mainCtrID, path string) error {
	outFile, err := os.Create(path)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer func() { _ = outFile.Close() }()
	reader, err := we.RuntimeExecutor.GetOutputStream(mainCtrID, true)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	_, err = io.Copy(outFile, reader)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

// InitDriver initializes an instance of an artifact driver
func (we *WorkflowExecutor) InitDriver(art wfv1.Artifact) (artifact.ArtifactDriver, error) {
	if art.S3 != nil {
		var accessKey string
		var secretKey string

		if art.S3.AccessKeySecret.Name != "" {
			accessKeyBytes, err := we.GetSecretFromVolMount(art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = string(accessKeyBytes)
			secretKeyBytes, err := we.GetSecretFromVolMount(art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = string(secretKeyBytes)
		}

		driver := s3.S3ArtifactDriver{
			Endpoint:  art.S3.Endpoint,
			AccessKey: accessKey,
			SecretKey: secretKey,
			Secure:    art.S3.Insecure == nil || *art.S3.Insecure == false,
			Region:    art.S3.Region,
		}
		return &driver, nil
	}
	if art.HTTP != nil {
		return &http.HTTPArtifactDriver{}, nil
	}
	if art.Git != nil {
		gitDriver := git.GitArtifactDriver{
			InsecureIgnoreHostKey: art.Git.InsecureIgnoreHostKey,
		}
		if art.Git.UsernameSecret != nil {
			usernameBytes, err := we.GetSecretFromVolMount(art.Git.UsernameSecret.Name, art.Git.UsernameSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Username = string(usernameBytes)
		}
		if art.Git.PasswordSecret != nil {
			passwordBytes, err := we.GetSecretFromVolMount(art.Git.PasswordSecret.Name, art.Git.PasswordSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Password = string(passwordBytes)
		}
		if art.Git.SSHPrivateKeySecret != nil {
			sshPrivateKeyBytes, err := we.GetSecretFromVolMount(art.Git.SSHPrivateKeySecret.Name, art.Git.SSHPrivateKeySecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.SSHPrivateKey = string(sshPrivateKeyBytes)
		}

		return &gitDriver, nil
	}
	if art.Artifactory != nil {
		usernameBytes, err := we.GetSecretFromVolMount(art.Artifactory.UsernameSecret.Name, art.Artifactory.UsernameSecret.Key)
		if err != nil {
			return nil, err
		}
		passwordBytes, err := we.GetSecretFromVolMount(art.Artifactory.PasswordSecret.Name, art.Artifactory.PasswordSecret.Key)
		if err != nil {
			return nil, err
		}
		driver := artifactory.ArtifactoryArtifactDriver{
			Username: string(usernameBytes),
			Password: string(passwordBytes),
		}
		return &driver, nil

	}
	if art.HDFS != nil {
		return hdfs.CreateDriver(we, art.HDFS)
	}
	if art.Raw != nil {
		return &raw.RawArtifactDriver{}, nil
	}

	return nil, errors.Errorf(errors.CodeBadRequest, "Unsupported artifact driver for %s", art.Name)
}

// getPod is a wrapper around the pod interface to get the current pod from kube API server
func (we *WorkflowExecutor) getPod() (*apiv1.Pod, error) {
	podsIf := we.ClientSet.CoreV1().Pods(we.Namespace)
	var pod *apiv1.Pod
	var err error
	_ = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		pod, err = podsIf.Get(we.PodName, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get pod '%s': %v", we.PodName, err)
			if !retry.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return pod, nil
}

// GetNamespace returns the namespace
func (we *WorkflowExecutor) GetNamespace() string {
	return we.Namespace
}

// GetConfigMapKey retrieves a configmap value and memoizes the result
func (we *WorkflowExecutor) GetConfigMapKey(namespace, name, key string) (string, error) {
	cachedKey := fmt.Sprintf("%s/%s/%s", namespace, name, key)
	if val, ok := we.memoizedConfigMaps[cachedKey]; ok {
		return val, nil
	}
	configmapsIf := we.ClientSet.CoreV1().ConfigMaps(namespace)
	var configmap *apiv1.ConfigMap
	var err error
	_ = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		configmap, err = configmapsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get configmap '%s': %v", name, err)
			if !retry.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	// memoize all keys in the configmap since it's highly likely we will need to get a
	// subsequent key in the configmap (e.g. username + password) and we can save an API call
	for k, v := range configmap.Data {
		we.memoizedConfigMaps[fmt.Sprintf("%s/%s/%s", namespace, name, k)] = v
	}
	val, ok := we.memoizedConfigMaps[cachedKey]
	if !ok {
		return "", errors.Errorf(errors.CodeBadRequest, "configmap '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// GetSecrets retrieves a secret value and memoizes the result
func (we *WorkflowExecutor) GetSecrets(namespace, name, key string) ([]byte, error) {
	cachedKey := fmt.Sprintf("%s/%s/%s", namespace, name, key)
	if val, ok := we.memoizedSecrets[cachedKey]; ok {
		return val, nil
	}
	secretsIf := we.ClientSet.CoreV1().Secrets(namespace)
	var secret *apiv1.Secret
	var err error
	_ = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		secret, err = secretsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get secret '%s': %v", name, err)
			if !retry.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return []byte{}, errors.InternalWrapError(err)
	}
	// memoize all keys in the secret since it's highly likely we will need to get a
	// subsequent key in the secret (e.g. username + password) and we can save an API call
	for k, v := range secret.Data {
		we.memoizedSecrets[fmt.Sprintf("%s/%s/%s", namespace, name, k)] = v
	}
	val, ok := we.memoizedSecrets[cachedKey]
	if !ok {
		return []byte{}, errors.Errorf(errors.CodeBadRequest, "secret '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// GetMainContainerStatus returns the container status of the main container, nil if the main container does not exist
func (we *WorkflowExecutor) GetMainContainerStatus() (*apiv1.ContainerStatus, error) {
	pod, err := we.getPod()
	if err != nil {
		return nil, err
	}
	for _, ctrStatus := range pod.Status.ContainerStatuses {
		if ctrStatus.Name == common.MainContainerName {
			return &ctrStatus, nil
		}
	}
	return nil, nil
}

// GetMainContainerID returns the container id of the main container
func (we *WorkflowExecutor) GetMainContainerID() (string, error) {
	if we.mainContainerID != "" {
		return we.mainContainerID, nil
	}
	ctrStatus, err := we.GetMainContainerStatus()
	if err != nil {
		return "", err
	}
	if ctrStatus == nil {
		return "", nil
	}
	we.mainContainerID = containerID(ctrStatus.ContainerID)
	return we.mainContainerID, nil
}

// CaptureScriptResult will add the stdout of a script template as output result
func (we *WorkflowExecutor) CaptureScriptResult() error {
	if we.Template.Script == nil {
		return nil
	}
	log.Infof("Capturing script output")
	mainContainerID, err := we.GetMainContainerID()
	if err != nil {
		return err
	}
	reader, err := we.RuntimeExecutor.GetOutputStream(mainContainerID, false)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	out := string(bytes)
	// Trims off a single newline for user convenience
	outputLen := len(out)
	if outputLen > 0 && out[outputLen-1] == '\n' {
		out = out[0 : outputLen-1]
	}
	we.Template.Outputs.Result = &out
	return nil
}

// AnnotateOutputs annotation to the pod indicating all the outputs.
func (we *WorkflowExecutor) AnnotateOutputs(logArt *wfv1.Artifact) error {
	outputs := we.Template.Outputs.DeepCopy()
	if logArt != nil {
		outputs.Artifacts = append(outputs.Artifacts, *logArt)
	}

	if !outputs.HasOutputs() {
		return nil
	}
	log.Infof("Annotating pod with output")
	outputBytes, err := json.Marshal(outputs)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return we.AddAnnotation(common.AnnotationKeyOutputs, string(outputBytes))
}

// AddError adds an error to the list of encountered errors durign execution
func (we *WorkflowExecutor) AddError(err error) {
	log.Errorf("executor error: %+v", err)
	we.errors = append(we.errors, err)
}

// AddAnnotation adds an annotation to the workflow pod
func (we *WorkflowExecutor) AddAnnotation(key, value string) error {
	return common.AddPodAnnotation(we.ClientSet, we.PodName, we.Namespace, key, value)
}

// isTarball returns whether or not the file is a tarball
func isTarball(filePath string) bool {
	cmd := exec.Command("tar", "-tf", filePath)
	log.Info(cmd.Args)
	err := cmd.Run()
	return err == nil
}

// untar extracts a tarball to a temporary directory,
// renaming it to the desired location
func untar(tarPath string, destPath string) error {
	// first extract the tar into a temporary dir
	tmpDir := destPath + ".tmpdir"
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = common.RunCommand("tar", "-xf", tarPath, "-C", tmpDir)
	if err != nil {
		return err
	}
	// next, decide how we wish to rename the file/dir
	// to the destination path.
	files, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	if len(files) == 1 {
		// if the tar is comprised of single file or directory,
		// rename that file to the desired location
		filePath := path.Join(tmpDir, files[0].Name())
		err = os.Rename(filePath, destPath)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		err = os.Remove(tmpDir)
		if err != nil {
			return errors.InternalWrapError(err)
		}
	} else {
		// the tar extracted into multiple files. In this case,
		// just rename the temp directory to the dest path
		err = os.Rename(tmpDir, destPath)
		if err != nil {
			return errors.InternalWrapError(err)
		}
	}
	return nil
}

// containerID is a convenience function to strip the 'docker://', 'containerd://' from k8s ContainerID string
func containerID(ctrID string) string {
	schemeIndex := strings.Index(ctrID, "://")
	if schemeIndex == -1 {
		return ctrID
	}
	return ctrID[schemeIndex+3:]
}

// Wait is the sidecar container logic which waits for the main container to complete.
// Also monitors for updates in the pod annotations which may change (e.g. terminate)
// Upon completion, kills any sidecars after it finishes.
func (we *WorkflowExecutor) Wait() error {
	err := we.RuntimeExecutor.WaitInit()
	if err != nil {
		return err
	}
	log.Infof("Waiting on main container")
	mainContainerID, err := we.waitMainContainerStart()
	if err != nil {
		return err
	}
	log.Infof("main container started with container ID: %s", mainContainerID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	annotationUpdatesCh := we.monitorAnnotations(ctx)
	go we.monitorDeadline(ctx, annotationUpdatesCh)

	err = we.RuntimeExecutor.Wait(mainContainerID)
	if err != nil {
		return err
	}
	log.Infof("Main container completed")
	return nil
}

// waitMainContainerStart waits for the main container to start and returns its container ID.
func (we *WorkflowExecutor) waitMainContainerStart() (string, error) {
	for {
		podsIf := we.ClientSet.CoreV1().Pods(we.Namespace)
		fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", we.PodName))
		opts := metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		}
		watchIf, err := podsIf.Watch(opts)
		if err != nil {
			return "", errors.InternalWrapErrorf(err, "Failed to establish pod watch: %v", err)
		}
		for watchEv := range watchIf.ResultChan() {
			if watchEv.Type == watch.Error {
				return "", errors.InternalErrorf("Pod watch error waiting for main to start: %v", watchEv.Object)
			}
			pod, ok := watchEv.Object.(*apiv1.Pod)
			if !ok {
				log.Warnf("Pod watch returned non pod object: %v", watchEv.Object)
				continue
			}
			for _, ctrStatus := range pod.Status.ContainerStatuses {
				if ctrStatus.Name == common.MainContainerName {
					log.Debug(ctrStatus)
					if ctrStatus.ContainerID != "" {
						we.mainContainerID = containerID(ctrStatus.ContainerID)
						return containerID(ctrStatus.ContainerID), nil
					} else if ctrStatus.State.Waiting == nil && ctrStatus.State.Running == nil && ctrStatus.State.Terminated == nil {
						// status still not ready, wait
					} else if ctrStatus.State.Waiting != nil {
						// main container is still in waiting status
					} else {
						// main container in running or terminated state but missing container ID
						return "", errors.InternalError("Main container ID cannot be found")
					}
				}
			}
		}
		log.Warnf("Pod watch closed unexpectedly")
	}
}

func watchFileChanges(ctx context.Context, pollInterval time.Duration, filePath string) <-chan struct{} {
	res := make(chan struct{})
	go func() {
		defer close(res)

		var modTime *time.Time
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			file, err := os.Stat(filePath)
			if err != nil {
				log.Fatal(err)
			}
			newModTime := file.ModTime()
			if modTime != nil && !modTime.Equal(file.ModTime()) {
				res <- struct{}{}
			}
			modTime = &newModTime
			time.Sleep(pollInterval)
		}
	}()
	return res
}

// monitorAnnotations starts a goroutine which monitors for any changes to the pod annotations.
// Emits an event on the returned channel upon any updates
func (we *WorkflowExecutor) monitorAnnotations(ctx context.Context) <-chan struct{} {
	log.Infof("Starting annotations monitor")

	// Create a channel to listen for a SIGUSR2. Upon receiving of the signal, we force reload our annotations
	// directly from kubernetes API. The controller uses this to fast-track notification of annotations
	// instead of waiting for the volume file to get updated (which can take minutes)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)

	we.setExecutionControl()

	// Create a channel which will notify a listener on new updates to the annotations
	annotationUpdateCh := make(chan struct{})

	annotationChanges := watchFileChanges(ctx, 10*time.Second, we.PodAnnotationsPath)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Infof("Annotations monitor stopped")
				signal.Stop(sigs)
				close(sigs)
				close(annotationUpdateCh)
				return
			case <-sigs:
				log.Infof("Received update signal. Reloading annotations from API")
				annotationUpdateCh <- struct{}{}
				we.setExecutionControl()
			case <-annotationChanges:
				log.Infof("%s updated", we.PodAnnotationsPath)
				err := we.LoadExecutionControl()
				if err != nil {
					log.Warnf("Failed to reload execution control from annotations: %v", err)
					continue
				}
				if we.ExecutionControl != nil {
					log.Infof("Execution control reloaded from annotations: %v", *we.ExecutionControl)
				}
				annotationUpdateCh <- struct{}{}
			}
		}
	}()
	return annotationUpdateCh
}

// setExecutionControl sets the execution control information from the pod annotation
func (we *WorkflowExecutor) setExecutionControl() {
	pod, err := we.getPod()
	if err != nil {
		log.Warnf("Failed to set execution control from API server: %v", err)
		return
	}
	execCtlString, ok := pod.ObjectMeta.Annotations[common.AnnotationKeyExecutionControl]
	if !ok {
		we.ExecutionControl = nil
	} else {
		var execCtl common.ExecutionControl
		err = json.Unmarshal([]byte(execCtlString), &execCtl)
		if err != nil {
			log.Errorf("Error unmarshalling '%s': %v", execCtlString, err)
			return
		}
		we.ExecutionControl = &execCtl
		log.Infof("Execution control set from API: %v", *we.ExecutionControl)
	}
}

// monitorDeadline checks to see if we exceeded the deadline for the step and
// terminates the main container if we did
func (we *WorkflowExecutor) monitorDeadline(ctx context.Context, annotationsUpdate <-chan struct{}) {
	log.Infof("Starting deadline monitor")
	for {
		select {
		case <-ctx.Done():
			log.Info("Deadline monitor stopped")
			return
		case <-annotationsUpdate:
		default:
			// TODO(jessesuen): we do not effectively use the annotations update channel yet. Ideally, we
			// should optimize this logic so that we use some type of mutable timer against the deadline
			// value instead of polling.
			if we.ExecutionControl != nil && we.ExecutionControl.Deadline != nil {
				if time.Now().UTC().After(*we.ExecutionControl.Deadline) {
					var message string
					// Zero value of the deadline indicates an intentional cancel vs. a timeout. We treat
					// timeouts as a failure and the pod should be annotated with that error
					if we.ExecutionControl.Deadline.IsZero() {
						message = "terminated"
					} else {
						message = fmt.Sprintf("step exceeded workflow deadline %s", *we.ExecutionControl.Deadline)
					}
					log.Info(message)
					_ = we.AddAnnotation(common.AnnotationKeyNodeMessage, message)
					log.Infof("Killing main container")
					mainContainerID, _ := we.GetMainContainerID()
					err := we.RuntimeExecutor.Kill([]string{mainContainerID})
					if err != nil {
						log.Warnf("Failed to kill main container: %v", err)
					}
					return
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// KillSidecars kills any sidecars to the main container
func (we *WorkflowExecutor) KillSidecars() error {
	if len(we.Template.Sidecars) == 0 {
		log.Infof("No sidecars")
		return nil
	}
	log.Infof("Killing sidecars")
	pod, err := we.getPod()
	if err != nil {
		return err
	}
	sidecarIDs := make([]string, 0)
	for _, ctrStatus := range pod.Status.ContainerStatuses {
		if ctrStatus.Name == common.MainContainerName || ctrStatus.Name == common.WaitContainerName {
			continue
		}
		if ctrStatus.State.Terminated != nil {
			continue
		}
		containerID := containerID(ctrStatus.ContainerID)
		log.Infof("Killing sidecar %s (%s)", ctrStatus.Name, containerID)
		sidecarIDs = append(sidecarIDs, containerID)
	}
	if len(sidecarIDs) == 0 {
		return nil
	}
	return we.RuntimeExecutor.Kill(sidecarIDs)
}

// LoadExecutionControl reads the execution control definition from the the Kubernetes downward api annotations volume file
func (we *WorkflowExecutor) LoadExecutionControl() error {
	err := unmarshalAnnotationField(we.PodAnnotationsPath, common.AnnotationKeyExecutionControl, &we.ExecutionControl)
	if err != nil {
		if errors.IsCode(errors.CodeNotFound, err) {
			return nil
		}
		return err
	}
	return nil
}

// LoadTemplate reads the template definition from the the Kubernetes downward api annotations volume file
func LoadTemplate(path string) (*wfv1.Template, error) {
	var tmpl wfv1.Template
	err := unmarshalAnnotationField(path, common.AnnotationKeyTemplate, &tmpl)
	if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// unmarshalAnnotationField unmarshals the value of an annotation key into the supplied interface
// from the downward api annotation volume file
func unmarshalAnnotationField(filePath string, key string, into interface{}) error {
	// Read the annotation file
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("ERROR opening annotation file from %s", filePath)
		return errors.InternalWrapError(err)
	}

	defer func() {
		_ = file.Close()
	}()
	reader := bufio.NewReader(file)

	// Prefix of key property in the annotation file
	prefix := fmt.Sprintf("%s=", key)

	for {
		// Read line-by-line
		var buffer bytes.Buffer
		var l []byte
		var isPrefix bool
		for {
			l, isPrefix, err = reader.ReadLine()
			buffer.Write(l)
			// If we've reached the end of the line, stop reading.
			if !isPrefix {
				break
			}
			// If we're just at the EOF, break
			if err != nil {
				break
			}
		}

		line := buffer.String()

		// Read property
		if strings.HasPrefix(line, prefix) {
			// Trim the prefix
			content := strings.TrimPrefix(line, prefix)

			// This part is a bit tricky in terms of unmarshalling
			// The content in the file will be something like,
			// `"{\"type\":\"container\",\"inputs\":{},\"outputs\":{}}"`
			// which is required to unmarshal twice

			// First unmarshal to a string without escaping characters
			var fieldString string
			err = json.Unmarshal([]byte(content), &fieldString)
			if err != nil {
				log.Errorf("Error unmarshalling annotation into string, %s, %v\n", content, err)
				return errors.InternalWrapError(err)
			}

			// Second unmarshal to a template
			err = json.Unmarshal([]byte(fieldString), into)
			if err != nil {
				log.Errorf("Error unmarshalling annotation into datastructure, %s, %v\n", fieldString, err)
				return errors.InternalWrapError(err)
			}
			return nil
		}

		// The end of the annotation file
		if err == io.EOF {
			break
		}
	}

	if err != io.EOF {
		return errors.InternalWrapError(err)
	}

	// If we reach here, then the key does not exist in the file
	return errors.Errorf(errors.CodeNotFound, "Key %s not found in annotation file: %s", key, filePath)
}
