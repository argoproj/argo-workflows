package executor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/artifacts/git"
	"github.com/argoproj/argo/workflow/artifacts/http"
	"github.com/argoproj/argo/workflow/artifacts/s3"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WorkflowExecutor implements the mechanisms within a single Kubernetes pod
type WorkflowExecutor struct {
	PodName   string
	Template  wfv1.Template
	ClientSet *kubernetes.Clientset
	Namespace string

	// memoized container ID to prevent multiple lookups
	mainContainerID string
}

// Use Kubernetes client to retrieve the Kubernetes secrets
func (we *WorkflowExecutor) getSecrets(namespace string, name string, key string) (string, error) {
	secrets, err := we.ClientSet.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		return "", errors.InternalWrapError(err)
	}

	val, ok := secrets.Data[key]
	if !ok {
		return "", errors.InternalErrorf("Key %s does not exists for secret %s", key, name)
	}
	return string(val), nil
}

func (we *WorkflowExecutor) LoadArtifacts() error {
	log.Infof("Start loading input artifacts...")

	for _, art := range we.Template.Inputs.Artifacts {
		log.Infof("Downloading artifact: %s", art.Name)
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
			artPath = path.Join(common.InitContainerMainFilesystemDir, art.Path)
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

// LoadScriptSource will create the script source file used in script containers
func (we *WorkflowExecutor) LoadScriptSource() error {
	if we.Template.Script == nil {
		return nil
	}
	log.Infof("Loading script source to %s", common.ScriptTemplateSourcePath)
	source := []byte(we.Template.Script.Source)
	err := ioutil.WriteFile(common.ScriptTemplateSourcePath, source, 0644)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

// SaveArtifacts uploads artifacts to the archive location
func (we *WorkflowExecutor) SaveArtifacts() error {
	log.Infof("Saving output artifacts")
	if len(we.Template.Outputs.Artifacts) == 0 {
		log.Infof("No output artifacts, nothing to do")
		return nil
	}
	mainCtrID, err := we.GetMainContainerID()
	if err != nil {
		return err
	}
	log.Infof("Main container identified as: %s", mainCtrID)

	// This directory temporarily stores the tarballs of the artifacts before uploading
	tempOutArtDir := "/argo/outputs/artifacts"
	err = os.MkdirAll(tempOutArtDir, os.ModePerm)
	if err != nil {
		return errors.InternalWrapError(err)
	}

	for i, art := range we.Template.Outputs.Artifacts {
		log.Infof("Saving artifact: %s", art.Name)
		// Determine the file path of where to find the artifact
		if art.Path == "" {
			return errors.InternalErrorf("Artifact %s did not specify a path", art.Name)
		}

		fileName := fmt.Sprintf("%s.tgz", art.Name)
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
			} else {
				return errors.Errorf(errors.CodeBadRequest, "Unable to determine path to store %s. Archive location provided no information", art.Name)
			}
		}

		tempArtPath := path.Join(tempOutArtDir, fileName)
		err = archivePath(mainCtrID, art.Path, tempArtPath)
		if err != nil {
			return err
		}
		artDriver, err := we.InitDriver(art)
		if err != nil {
			return err
		}
		err = artDriver.Save(tempArtPath, &art)
		if err != nil {
			return err
		}
		// remove is best effort (the container will go away anyways).
		// we just want reduce peak space usage
		err = os.Remove(tempArtPath)
		if err != nil {
			log.Warn("Failed to remove %s", tempArtPath)
		}
		we.Template.Outputs.Artifacts[i] = art
		log.Infof("Successfully saved file: %s", tempArtPath)
	}
	return nil
}

// SaveParameters will save the content in the specified file path as output parameter value
func (we *WorkflowExecutor) SaveParameters() error {
	log.Infof("Saving output parameters")
	if len(we.Template.Outputs.Parameters) == 0 {
		log.Infof("No output parameters, nothing to do")
		return nil
	}
	mainCtrID, err := we.GetMainContainerID()
	if err != nil {
		return err
	}
	log.Infof("Main container identified as: %s", mainCtrID)

	for i, param := range we.Template.Outputs.Parameters {
		log.Infof("Saving out parameter: %s", param.Name)
		// Determine the file path of where to find the parameter
		if param.Path == "" {
			return errors.InternalErrorf("Output parameter %s did not specify a file path", param.Name)
		}
		// Use docker cp command to print out the content of the file
		// Node docker cp CONTAINER:SRC_PATH DEST_PATH|- streams the contents of the resource
		// as a tar archive to STDOUT if using - as DEST_PATH. Thus, we need to extract the
		// content from the tar archive and output into stdout. In this way, we do not need to
		// create and copy the content into a file from the wait container.
		dockerCpCmd := fmt.Sprintf("docker cp -a %s:%s - | tar -ax -O", mainCtrID, param.Path)
		cmd := exec.Command("sh", "-c", dockerCpCmd)
		log.Info(cmd.Args)
		out, err := cmd.Output()
		if err != nil {
			if exErr, ok := err.(*exec.ExitError); ok {
				log.Errorf("`%s` stderr:\n%s", cmd.Args, string(exErr.Stderr))
			}
			return errors.InternalWrapError(err)
		}
		output := string(out)
		we.Template.Outputs.Parameters[i].Value = &output
		log.Infof("Successfully saved output parameter: %s", param.Name)
	}
	return nil
}

func (we *WorkflowExecutor) InitDriver(art wfv1.Artifact) (artifact.ArtifactDriver, error) {
	if art.S3 != nil {
		// Getting Kubernetes namespace from the environment variables
		namespace := os.Getenv(common.EnvVarNamespace)
		accessKey, err := we.getSecrets(namespace, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
		if err != nil {
			return nil, err
		}
		secretKey, err := we.getSecrets(namespace, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
		if err != nil {
			return nil, err
		}
		driver := s3.S3ArtifactDriver{
			Endpoint:  art.S3.Endpoint,
			AccessKey: accessKey,
			SecretKey: secretKey,
			Secure:    art.S3.Insecure == nil || *art.S3.Insecure == false,
		}
		return &driver, nil
	}
	if art.HTTP != nil {
		return &http.HTTPArtifactDriver{}, nil
	}
	if art.Git != nil {
		return &git.GitArtifactDriver{}, nil
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unsupported artifact driver for %s", art.Name)
}

// GetMainContainerStatus returns the container status of the main container
func (we *WorkflowExecutor) GetMainContainerStatus() (*apiv1.ContainerStatus, error) {
	podIf := we.ClientSet.CoreV1().Pods(we.Namespace)
	pod, err := podIf.Get(we.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	for _, ctrStatus := range pod.Status.ContainerStatuses {
		if ctrStatus.Name == common.MainContainerName {
			return &ctrStatus, nil
		}
	}
	return nil, errors.InternalErrorf("Main container not found")
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
	mainCtrID := strings.Replace(ctrStatus.ContainerID, "docker://", "", 1)
	we.mainContainerID = mainCtrID
	return mainCtrID, nil
}

func archivePath(containerID string, sourcePath string, destPath string) error {
	log.Infof("Archiving %s:%s to %s", containerID, sourcePath, destPath)
	dockerCpCmd := fmt.Sprintf("docker cp -a %s:%s - | gzip > %s", containerID, sourcePath, destPath)
	err := common.RunCommand("sh", "-c", dockerCpCmd)
	if err != nil {
		return err
	}
	log.Infof("Archiving completed")
	return nil
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
	cmd := exec.Command("docker", "logs", mainContainerID)
	log.Info(cmd.Args)
	outBytes, _ := cmd.Output()
	outStr := strings.TrimSpace(string(outBytes))
	we.Template.Outputs.Result = &outStr
	return nil
}

// AnnotateOutputs annotation to the pod indicating all the outputs.
func (we *WorkflowExecutor) AnnotateOutputs() error {
	if !we.Template.Outputs.HasOutputs() {
		return nil
	}
	log.Infof("Annotating pod with output")
	outputBytes, err := json.Marshal(we.Template.Outputs)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return we.AddAnnotation(common.AnnotationKeyOutputs, string(outputBytes))
}

// AddAnnotation adds an annotation to the workflow pod
func (we *WorkflowExecutor) AddAnnotation(key, value string) error {
	// TODO: switch to k8s go-sdk to perform this logic.
	// See kubectl/cmd/annotate.go for reference implementation.
	// For now we just kubectl because it uses our desired
	// overwite Patch strategy.
	return common.RunCommand("kubectl", "annotate", "--overwrite", "pods",
		we.PodName, fmt.Sprintf("%s=%s", key, value))
}

// isTarball returns whether or not the file is a tarball
func isTarball(filePath string) bool {
	cmd := exec.Command("tar", "-tzf", filePath)
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

// containerID is a convenience function to strip the 'docker://' from k8s ContainerID string
func containerID(ctrID string) string {
	return strings.Replace(ctrID, "docker://", "", 1)
}

// Wait is the sidecar container waits for the main container to complete and kills any sidecars after it finishes
func (we *WorkflowExecutor) Wait() error {
	log.Infof("Waiting on main container")
	var mainContainerID string
	for {
		ctrStatus, err := we.GetMainContainerStatus()
		if err != nil {
			return err
		}
		log.Debug(ctrStatus)
		if ctrStatus.ContainerID != "" {
			mainContainerID = containerID(ctrStatus.ContainerID)
			break
		} else if ctrStatus.State.Waiting == nil && ctrStatus.State.Running == nil && ctrStatus.State.Terminated == nil {
			// status still not ready, wait
			time.Sleep(5)
		} else if ctrStatus.State.Waiting != nil {
			// main container is still in waiting status
			time.Sleep(5)
		} else {
			// main container in running or terminated state but missing container ID
			return errors.InternalErrorf("Container ID cannot be found")
		}
	}

	err := common.RunCommand("docker", "wait", mainContainerID)
	if err != nil {
		return err
	}
	log.Infof("Waiting completed")
	err = we.killSidecars()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

// killGracePeriod is the time in seconds after sending SIGTERM before
// forcefully killing the sidcar with SIGKILL (value matches k8s)
const killGracePeriod = 30

func (we *WorkflowExecutor) killSidecars() error {
	log.Infof("Killing sidecars")
	podIf := we.ClientSet.CoreV1().Pods(we.Namespace)
	pod, err := podIf.Get(we.PodName, metav1.GetOptions{})
	if err != nil {
		return errors.InternalWrapError(err)
	}
	sidecarIDs := make([]string, 0)
	killArgs := []string{"kill"}
	waitArgs := []string{"wait"}
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
		killArgs = append(killArgs, containerID)
		waitArgs = append(waitArgs, containerID)
	}
	if len(sidecarIDs) == 0 {
		return nil
	}
	killArgs = append(killArgs, "--signal", "TERM")
	err = common.RunCommand("docker", killArgs...)
	if err != nil {
		return err
	}

	log.Infof("Waiting (%ds) for sidecars to terminate", killGracePeriod)
	cmd := exec.Command("docker", waitArgs...)
	log.Info(cmd.Args)
	if err := cmd.Start(); err != nil {
		return errors.InternalWrapError(err)
	}
	timer := time.AfterFunc(killGracePeriod*time.Second, func() {
		log.Infof("Timed out (%ds) for sidecars to terminate gracefully. Killing forcefully", killGracePeriod)
		cmd.Process.Kill()
		killArgs[len(waitArgs)-1] = "KILL"
		cmd = exec.Command("docker", killArgs...)
		log.Info(cmd.Args)
		_ = cmd.Run()
	})
	err = cmd.Wait()
	timer.Stop()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return err
}
