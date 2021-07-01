package docker

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var errContainerNotExist = fmt.Errorf("container does not exist") // sentinel error

var started = time.Now()

type DockerExecutor struct {
	namespace  string
	podName    string
	containers map[string]ctr // containerName -> ctr
}

type ctr struct {
	containerID string
	status      string
	createdAt   time.Time
}

func NewDockerExecutor(namespace, podName string) (*DockerExecutor, error) {
	log.Infof("Creating a docker executor")
	return &DockerExecutor{namespace, podName, make(map[string]ctr)}, nil
}

func (d *DockerExecutor) GetFileContents(containerName string, sourcePath string) (string, error) {
	// Uses docker cp command to return contents of the file
	// NOTE: docker cp CONTAINER:SRC_PATH DEST_PATH|- streams the contents of the resource
	// as a tar archive to STDOUT if using - as DEST_PATH. Thus, we need to extract the
	// content from the tar archive and output into stdout. In this way, we do not need to
	// create and copy the content into a file from the wait container.
	containerID, err := d.getContainerID(containerName)
	if err != nil {
		return "", err
	}
	dockerCpCmd := fmt.Sprintf("docker cp -a %s:%s - | tar -ax -O", containerID, sourcePath)
	out, err := common.RunShellCommand(dockerCpCmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (d *DockerExecutor) CopyFile(containerName string, sourcePath string, destPath string, compressionLevel int) error {
	log.Infof("Archiving %s:%s to %s", containerName, sourcePath, destPath)
	containerID, err := d.getContainerID(containerName)
	if err != nil {
		return err
	}
	dockerCpCmd := getDockerCpCmd(containerID, sourcePath, compressionLevel, destPath)
	_, err = common.RunShellCommand(dockerCpCmd)
	if err != nil {
		return err
	}
	copiedFile, err := os.Open(filepath.Clean(destPath))
	if err != nil {
		return err
	}
	defer util.Close(copiedFile)
	gzipReader, err := file.GetGzipReader(copiedFile)
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

func (d *DockerExecutor) GetOutputStream(ctx context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error) {
	containerID, err := d.getContainerID(containerName)
	if err != nil {
		return nil, err
	}
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

func (d *DockerExecutor) GetExitCode(ctx context.Context, containerName string) (string, error) {
	containerID, err := d.getContainerID(containerName)
	if err != nil {
		return "", err
	}
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

func (d *DockerExecutor) Wait(ctx context.Context, containerNames []string) error {
	go func() {
		err := d.pollContainerIDs(ctx, containerNames)
		if err != nil {
			log.WithError(err).Error("failed to poll container IDs")
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if d.haveContainers(containerNames) {
				containerIDs, err := d.getContainerIDs(containerNames)
				if err != nil {
					return err
				}
				_, err = common.RunCommand("docker", append([]string{"wait"}, containerIDs...)...)
				if err != nil && strings.Contains(err.Error(), "No such container") {
					// e.g. reason could be ContainerCannotRun
					log.WithError(err).Info("ignoring error as container may have been re-created and therefore container ID may have changed")
					continue
				}
				return err
			}
			time.Sleep(time.Second)
		}
	}
}

func (d *DockerExecutor) listContainers() (map[string]ctr, error) {
	output, err := common.RunCommand(
		"docker",
		"ps",
		"--all",      // container could have already exited, but there could also have been two containers for the same pod (old container not yet cleaned-up)
		"--no-trunc", // display long container IDs
		"--format={{.Status}}|{{.Label \"io.kubernetes.container.name\"}}|{{.ID}}|{{.CreatedAt}}", // similar to `Up 3 hours,main,035a98c4e72e,2021-03-08 17:25:15 -0800 PST`
		// https://github.com/kubernetes/kubernetes/blob/ca6bdba014f0a98efe0e0dd4e15f57d1c121d6c9/pkg/kubelet/dockertools/labels.go#L37
		"--filter=label=io.kubernetes.pod.namespace="+d.namespace,
		"--filter=label=io.kubernetes.pod.name="+d.podName,
	)
	if err != nil {
		return nil, err
	}
	containers := make(map[string]ctr)
	for _, l := range strings.Split(string(output), "\n") {
		parts := strings.Split(strings.TrimSpace(l), "|")
		if len(parts) != 4 {
			continue
		}
		status := strings.SplitN(parts[0], " ", 2)[0] // Created,Exited,Up,
		containerName := parts[1]
		if containerName == "POD" {
			continue
		}
		containerID := parts[2]
		if containerID == "" {
			continue
		}
		createdAt, err := time.Parse("2006-01-02 15:04:05 -0700 MST", parts[3])
		if err != nil {
			return nil, err
		}
		containers[containerName] = ctr{containerID: containerID, status: status, createdAt: createdAt}
	}
	log.WithField("containers", containers).Info("listed containers")
	return containers, nil
}

func (d *DockerExecutor) pollContainerIDs(ctx context.Context, containerNames []string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			containers, err := d.listContainers()
			if err != nil {
				return err
			}
			for containerName, c := range containers {
				if d.containers[containerName].containerID == c.containerID { // already found
					continue
				}
				if c.createdAt.Before(started.Add(-15 * time.Second)) {
					log.Infof("ignoring container %q created at %v, too long before process started", containerName, c.createdAt)
					continue
				}
				d.containers[containerName] = c
				log.Infof("mapped container name %q to container ID %q (created at %v, status %s)", containerName, c.containerID, c.createdAt, c.status)
			}
		}
		// sidecars start after the main containers, so we can't just exit once we know about all the main containers,
		// we need a bit more time
		if !d.haveContainers(containerNames) || time.Since(started) < 30*time.Second {
			time.Sleep(1 * time.Second) // this is a hard-loop because containers can run very short periods of time
		} else {
			time.Sleep(10 * time.Second)
		}
	}
}

func (d *DockerExecutor) haveContainers(containerNames []string) bool {
	for _, n := range containerNames {
		if _, ok := d.containers[n]; !ok {
			return false
		}
	}
	return true
}

func (d *DockerExecutor) getContainerID(containerName string) (string, error) {
	if ctr, ok := d.containers[containerName]; ok {
		return ctr.containerID, nil
	}
	return "", errContainerNotExist
}

// killContainers kills a list of containerNames first with a SIGTERM then with a SIGKILL after a grace period
func (d *DockerExecutor) Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	containerIDs, err := d.getContainerIDs(containerNames)
	if err != nil {
		return err
	}

	if len(containerIDs) == 0 { // they may have already terminated
		log.Info("zero container IDs, assuming all containers have exited successfully")
		return nil
	}

	killArgs := append([]string{"kill", "--signal", "TERM"}, containerIDs...)
	// docker kill will return with an error if a container has terminated already, which is not an error in this case.
	// We therefore ignore any error. docker wait that follows will re-raise any other error with the container.
	_, err = common.RunCommand("docker", killArgs...)
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
	// Otherwise, if the terminationGracePeriodSeconds elapses and the forced kill branch is run, there would
	// be no receiver for waitCh and the goroutine would block forever
	waitCh := make(chan error, 1)
	go func() {
		defer close(waitCh)
		waitCh <- waitCmd.Wait()
	}()
	select {
	case err = <-waitCh:
		// waitCmd completed
	case <-time.After(terminationGracePeriodDuration):
		log.Infof("Timed out (%v) for containers to terminate gracefully. Killing forcefully", terminationGracePeriodDuration)
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

func (d *DockerExecutor) ListContainerNames(ctx context.Context) ([]string, error) {
	containers, err := d.listContainers()
	if err != nil {
		return nil, err
	}
	var containerNames []string
	for n := range containers {
		containerNames = append(containerNames, n)
	}
	return containerNames, nil
}

func (d *DockerExecutor) getContainerIDs(containerNames []string) ([]string, error) {
	var containerIDs []string
	for _, n := range containerNames {
		containerID, err := d.getContainerID(n)
		if err == errContainerNotExist {
			continue
		}
		if err != nil {
			return nil, err
		}
		containerIDs = append(containerIDs, containerID)
	}
	return containerIDs, nil
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
