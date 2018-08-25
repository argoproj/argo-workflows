package docker

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
)

// killGracePeriod is the time in seconds after sending SIGTERM before
// forcefully killing the sidecar with SIGKILL (value matches k8s)
const killGracePeriod = 30

type DockerExecutor struct{}

func NewDockerExecutor() (*DockerExecutor, error) {
	log.Infof("Creating a docker executor")
	return &DockerExecutor{}, nil
}

func (d *DockerExecutor) GetFileContents(containerID string, sourcePath string) (string, error) {
	// Uses docker cp command to return contents of the file
	// NOTE: docker cp CONTAINER:SRC_PATH DEST_PATH|- streams the contents of the resource
	// as a tar archive to STDOUT if using - as DEST_PATH. Thus, we need to extract the
	// content from the tar archive and output into stdout. In this way, we do not need to
	// create and copy the content into a file from the wait container.
	dockerCpCmd := fmt.Sprintf("docker cp -a %s:%s - | tar -ax -O", containerID, sourcePath)
	cmd := exec.Command("sh", "-c", dockerCpCmd)
	log.Info(cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		if exErr, ok := err.(*exec.ExitError); ok {
			log.Errorf("`%s` stderr:\n%s", cmd.Args, string(exErr.Stderr))
		}
		return "", errors.InternalWrapError(err)
	}
	return string(out), nil
}

func (d *DockerExecutor) CopyFile(containerID string, sourcePath string, destPath string) error {
	log.Infof("Archiving %s:%s to %s", containerID, sourcePath, destPath)
	dockerCpCmd := fmt.Sprintf("docker cp -a %s:%s - | gzip > %s", containerID, sourcePath, destPath)
	err := common.RunCommand("sh", "-c", dockerCpCmd)
	if err != nil {
		return err
	}
	log.Infof("Archiving completed")
	return nil
}

// GetOutput returns the entirety of the container output as a string
// Used to capturing script results as an output parameter
func (d *DockerExecutor) GetOutput(containerID string) (string, error) {
	cmd := exec.Command("docker", "logs", containerID)
	log.Info(cmd.Args)
	outBytes, _ := cmd.Output()
	return strings.TrimSpace(string(outBytes)), nil
}

// Wait for the container to complete
func (d *DockerExecutor) Wait(containerID string) error {
	return common.RunCommand("docker", "wait", containerID)
}

// killContainers kills a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (d *DockerExecutor) Kill(containerIDs []string) error {
	killArgs := append([]string{"kill", "--signal", "TERM"}, containerIDs...)
	err := common.RunCommand("docker", killArgs...)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	waitArgs := append([]string{"wait"}, containerIDs...)
	waitCmd := exec.Command("docker", waitArgs...)
	log.Info(waitCmd.Args)
	if err := waitCmd.Start(); err != nil {
		return errors.InternalWrapError(err)
	}
	timer := time.AfterFunc(killGracePeriod*time.Second, func() {
		log.Infof("Timed out (%ds) for containers to terminate gracefully. Killing forcefully", killGracePeriod)
		_ = waitCmd.Process.Kill()
		forceKillArgs := append([]string{"kill", "--signal", "KILL"}, containerIDs...)
		forceKillCmd := exec.Command("docker", forceKillArgs...)
		log.Info(forceKillCmd.Args)
		_ = forceKillCmd.Run()
	})
	err = waitCmd.Wait()
	_ = timer.Stop()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Infof("Containers %s killed successfully", containerIDs)
	return nil
}
