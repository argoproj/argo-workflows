package docker

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/util/file"
	"github.com/argoproj/argo/workflow/common"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
)

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
	out, err := common.RunShellCommand(dockerCpCmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (d *DockerExecutor) CopyFile(containerID string, sourcePath string, destPath string, compressionLevel int) error {
	log.Infof("Archiving %s:%s to %s", containerID, sourcePath, destPath)

	dockerCpCmd := getDockerCpCmd(containerID, sourcePath, compressionLevel, destPath)
	_, err := common.RunShellCommand(dockerCpCmd)
	if err != nil {
		return err
	}
	copiedFile, err := os.Open(destPath)
	if err != nil {
		return err
	}
	defer util.Close(copiedFile)
	gzipReader, err := gzip.NewReader(copiedFile)
	if err != nil {
		return err
	}
	if !file.ExistsInTar(sourcePath, tar.NewReader(gzipReader)) {
		errMsg := fmt.Sprintf("path %s does not exist in archive %s", sourcePath, destPath)
		log.Warn(errMsg)
		return errors.Errorf(errors.CodeNotFound, errMsg)
	}
	log.Infof("Archiving completed")
	return nil
}

type cmdCloser struct {
	io.Reader
	cmd *exec.Cmd
}

func (c *cmdCloser) Close() error {
	err := c.cmd.Wait()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

func (d *DockerExecutor) GetOutputStream(containerID string, combinedOutput bool) (io.ReadCloser, error) {
	cmd := exec.Command("docker", "logs", containerID)
	log.Info(cmd.Args)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	if !combinedOutput {
		err = cmd.Start()
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		return stdout, nil
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	reader, writer := io.Pipe()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(writer, stdout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(writer, stderr)
	}()

	go func() {
		defer writer.Close()
		wg.Wait()
	}()

	return &cmdCloser{Reader: reader, cmd: cmd}, nil
}

func (d *DockerExecutor) GetExitCode(containerID string) (string, error) {
	cmd := exec.Command("docker", "inspect", containerID, "--format='{{.State.ExitCode}}'")
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.InternalWrapError(err, "Could not pipe STDOUT")
	}
	err = cmd.Start()
	if err != nil {
		return "", errors.InternalWrapError(err, "Could not start command")
	}
	defer func() { _ = reader.Close() }()
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", errors.InternalWrapError(err, "Could not read from STDOUT")
	}
	out := string(bytes)

	// Trims off a single newline for user convenience
	outputLen := len(out)
	if outputLen > 0 && out[outputLen-1] == '\n' {
		out = out[:outputLen-1]
	}
	exitCode := strings.Trim(out, `'`)
	// Ensure exit code is an int
	if _, err := strconv.Atoi(exitCode); err != nil {
		log.Warningf("Was not able to parse exit code output '%s' as int: %s", exitCode, err)
		return "", nil
	}
	return exitCode, nil
}

func (d *DockerExecutor) WaitInit() error {
	return nil
}

// Wait for the container to complete
func (d *DockerExecutor) Wait(containerID string) error {
	_, err := common.RunCommand("docker", "wait", containerID)
	return err
}

// killContainers kills a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (d *DockerExecutor) Kill(containerIDs []string) error {
	killArgs := append([]string{"kill", "--signal", "TERM"}, containerIDs...)
	// docker kill will return with an error if a container has terminated already, which is not an error in this case.
	// We therefore ignore any error. docker wait that follows will re-raise any other error with the container.
	_, err := common.RunCommand("docker", killArgs...)
	if err != nil {
		log.Warningf("Ignored error from 'docker kill --signal TERM': %s", err)
	}
	waitArgs := append([]string{"wait"}, containerIDs...)
	waitCmd := exec.Command("docker", waitArgs...)
	log.Info(waitCmd.Args)
	if err := waitCmd.Start(); err != nil {
		return errors.InternalWrapError(err)
	}
	// waitCh needs buffer of 1 so it can always send the result of waitCmd.Wait() without blocking.
	// Otherwise, if the KillGracePeriod elapses and the forced kill branch is run, there would
	// be no receiver for waitCh and the goroutine would block forever
	waitCh := make(chan error, 1)
	go func() {
		defer close(waitCh)
		waitCh <- waitCmd.Wait()
	}()
	select {
	case err = <-waitCh:
		// waitCmd completed
	case <-time.After(execcommon.KillGracePeriod * time.Second):
		log.Infof("Timed out (%ds) for containers to terminate gracefully. Killing forcefully", execcommon.KillGracePeriod)
		forceKillArgs := append([]string{"kill", "--signal", "KILL"}, containerIDs...)
		forceKillCmd := exec.Command("docker", forceKillArgs...)
		log.Info(forceKillCmd.Args)
		// same as kill case above, we ignore any error
		if err := forceKillCmd.Run(); err != nil {
			log.Warningf("Ignored error from 'docker kill --signal KILL': %s", err)
		}
	}
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Infof("Containers %s killed successfully", containerIDs)
	return nil
}

// getDockerCpCmd uses os-specific code to run `docker cp` and gzip/7zip to copy gzipped data from another
// container.
func getDockerCpCmd(containerID, sourcePath string, compressionLevel int, destPath string) string {
	gzipCmd := "gzip %s > %s"
	levelFlagParam := "-"
	if runtime.GOOS == "windows" {
		gzipCmd = "7za.exe a -tgzip -si %s %s"
		levelFlagParam = "-mx"
	}

	var levelFlag string
	switch compressionLevel {
	case gzip.NoCompression:
		// best we can do - if we skip gzip it's a different file
		levelFlag = levelFlagParam + "1"
	case gzip.DefaultCompression:
		// use cmd default
		levelFlag = ""
	default:
		// -1 through -9 (or error)
		levelFlag = levelFlagParam + strconv.Itoa(compressionLevel)
	}
	return fmt.Sprintf("docker cp -a %s:%s - | %s", containerID, sourcePath, fmt.Sprintf(gzipCmd, levelFlag, destPath))
}
