package executor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/artifacts/artifactory"
	"github.com/argoproj/argo/workflow/artifacts/git"
	"github.com/argoproj/argo/workflow/artifacts/http"
	"github.com/argoproj/argo/workflow/artifacts/s3"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
		return "", errors.Errorf(errors.CodeBadRequest, "secret '%s' does not have the key '%s'", name, key)
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
	log.Infof("Saving output artifacts")
	if len(we.Template.Outputs.Artifacts) == 0 {
		log.Infof("No output artifacts, nothing to do")
		return nil
	}
	mainCtrID, err := we.GetMainContainerID()
	if err != nil {
		return err
	}

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
			} else if we.Template.ArchiveLocation.Artifactory != nil {
				shallowCopy := *we.Template.ArchiveLocation.Artifactory
				art.Artifactory = &shallowCopy
				art.Artifactory.URL = path.Join(art.Artifactory.URL, fileName)
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
		accessKey, err := we.getSecrets(we.Namespace, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
		if err != nil {
			return nil, err
		}
		secretKey, err := we.getSecrets(we.Namespace, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
		if err != nil {
			return nil, err
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
		gitDriver := git.GitArtifactDriver{}
		if art.Git.UsernameSecret != nil {
			username, err := we.getSecrets(we.Namespace, art.Git.UsernameSecret.Name, art.Git.UsernameSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Username = username
		}
		if art.Git.PasswordSecret != nil {
			password, err := we.getSecrets(we.Namespace, art.Git.PasswordSecret.Name, art.Git.PasswordSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Password = password
		}

		return &gitDriver, nil
	}
	if art.Artifactory != nil {
		// Getting Kubernetes namespace from the environment variables
		namespace := os.Getenv(common.EnvVarNamespace)
		username, err := we.getSecrets(namespace, art.Artifactory.UsernameSecret.Name, art.Artifactory.UsernameSecret.Key)
		if err != nil {
			return nil, err
		}
		password, err := we.getSecrets(namespace, art.Artifactory.PasswordSecret.Name, art.Artifactory.PasswordSecret.Key)
		if err != nil {
			return nil, err
		}
		driver := artifactory.ArtifactoryArtifactDriver{
			Username: username,
			Password: password,
		}
		return &driver, nil

	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unsupported artifact driver for %s", art.Name)
}

// GetMainContainerStatus returns the container status of the main container, nil if the main container does not exist
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
	mainCtrID := strings.Replace(ctrStatus.ContainerID, "docker://", "", 1)
	we.mainContainerID = mainCtrID
	log.Infof("'main' container identified as: %s", mainCtrID)
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
	return common.AddPodAnnotation(we.ClientSet, we.PodName, we.Namespace, key, value)
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
		if ctrStatus != nil {
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
				return errors.InternalError("Container ID cannot be found")
			}
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
		_ = cmd.Process.Kill()
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

// ExecResource will run kubectl action against a manifest
func (we *WorkflowExecutor) ExecResource(action string, manifestPath string) (string, error) {
	args := []string{
		action,
	}
	if action == "delete" {
		args = append(args, "--ignore-not-found")
	}
	args = append(args, "-f")
	args = append(args, manifestPath)
	args = append(args, "-o")
	args = append(args, "name")
	cmd := exec.Command("kubectl", args...)
	log.Info(strings.Join(cmd.Args, " "))
	out, err := cmd.Output()
	if err != nil {
		exErr := err.(*exec.ExitError)
		errMsg := strings.TrimSpace(string(exErr.Stderr))
		return "", errors.New(errors.CodeBadRequest, errMsg)
	}
	resourceName := strings.TrimSpace(string(out))
	log.Infof(resourceName)
	return resourceName, nil
}

// gjsonLabels is an implementation of labels.Labels interface
// which allows us to take advantage of k8s labels library
// for the purposes of evaluating fail and success conditions
type gjsonLabels struct {
	json []byte
}

// Has returns whether the provided label exists.
func (g gjsonLabels) Has(label string) bool {
	return gjson.GetBytes(g.json, label).Exists()
}

// Get returns the value for the provided label.
func (g gjsonLabels) Get(label string) string {
	return gjson.GetBytes(g.json, label).String()
}

// WaitResource waits for a specific resource to satisfy either the success or failure condition
func (we *WorkflowExecutor) WaitResource(resourceName string) error {
	if we.Template.Resource.SuccessCondition == "" && we.Template.Resource.FailureCondition == "" {
		return nil
	}
	var successReqs labels.Requirements
	if we.Template.Resource.SuccessCondition != "" {
		successSelector, err := labels.Parse(we.Template.Resource.SuccessCondition)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "success condition '%s' failed to parse: %v", we.Template.Resource.SuccessCondition, err)
		}
		log.Infof("Waiting for conditions: %s", successSelector)
		successReqs, _ = successSelector.Requirements()
	}

	var failReqs labels.Requirements
	if we.Template.Resource.FailureCondition != "" {
		failSelector, err := labels.Parse(we.Template.Resource.FailureCondition)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "fail condition '%s' failed to parse: %v", we.Template.Resource.FailureCondition, err)
		}
		log.Infof("Failing for conditions: %s", failSelector)
		failReqs, _ = failSelector.Requirements()
	}

	cmd := exec.Command("kubectl", "get", resourceName, "-w", "-o", "json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	reader := bufio.NewReader(stdout)
	log.Info(strings.Join(cmd.Args, " "))
	if err := cmd.Start(); err != nil {
		return errors.InternalWrapError(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()
	for {
		jsonBytes, err := readJSON(reader)
		if err != nil {
			// TODO: it's possible that kubectl might EOF upon an API disconnect.
			// In this case, we would want to refork kubectl again.
			return errors.InternalWrapError(err)
		}
		log.Info(string(jsonBytes))
		ls := gjsonLabels{json: jsonBytes}
		for _, req := range failReqs {
			failed := req.Matches(ls)
			msg := fmt.Sprintf("failure condition '%s' evaluated %v", req, failed)
			log.Infof(msg)
			if failed {
				// TODO: need a better error code instead of BadRequest
				return errors.Errorf(errors.CodeBadRequest, msg)
			}
		}
		numMatched := 0
		for _, req := range successReqs {
			matched := req.Matches(ls)
			log.Infof("success condition '%s' evaluated %v", req, matched)
			if matched {
				numMatched++
			}
		}
		log.Infof("%d/%d success conditions matched", numMatched, len(successReqs))
		if numMatched >= len(successReqs) {
			break
		}
	}
	return nil
}

// readJSON reads from a reader line-by-line until it reaches "}\n" indicating end of json
func readJSON(reader *bufio.Reader) ([]byte, error) {
	var buffer bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		isDelimiter := len(line) == 2 && line[0] == byte('}')
		line = bytes.TrimSpace(line)
		_, err = buffer.Write(line)
		if err != nil {
			return nil, err
		}
		if isDelimiter {
			break
		}
	}
	return buffer.Bytes(), nil
}
