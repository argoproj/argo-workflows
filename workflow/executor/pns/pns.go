package pns

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	executil "github.com/argoproj/pkg/exec"
	gops "github.com/mitchellh/go-ps"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/v2/errors"
	"github.com/argoproj/argo/v2/util/archive"
	"github.com/argoproj/argo/v2/workflow/common"
	execcommon "github.com/argoproj/argo/v2/workflow/executor/common"
	"github.com/argoproj/argo/v2/workflow/executor/k8sapi"
	osspecific "github.com/argoproj/argo/v2/workflow/executor/os-specific"
)

var errContainerNameNotFound = fmt.Errorf("container name not found")

type PNSExecutor struct {
	*k8sapi.K8sAPIExecutor
	podName   string
	namespace string

	containers map[string]string // container name -> container ID

	// ctrIDToPid maps a containerID to a process ID
	ctrIDToPid map[string]int

	// pidFileHandles holds file handles to all root containers
	pidFileHandles map[int]*os.File

	// thisPID is the pid of this process
	thisPID int
	// rootFS holds a file descriptor to the root filesystem, allowing the executor to exit out of a chroot
	rootFS *os.File
}

func NewPNSExecutor(clientset *kubernetes.Clientset, podName, namespace string) (*PNSExecutor, error) {
	thisPID := os.Getpid()
	log.Infof("Creating PNS executor (namespace: %s, pod: %s, pid: %d)", namespace, podName, thisPID)
	if thisPID == 1 {
		return nil, errors.New(errors.CodeBadRequest, "process namespace sharing is not enabled on pod")
	}
	delegate, err := k8sapi.NewK8sAPIExecutor(clientset, nil, podName, namespace)
	if err != nil {
		return nil, err
	}
	return &PNSExecutor{
		K8sAPIExecutor: delegate,
		podName:        podName,
		namespace:      namespace,
		thisPID:        thisPID,
		ctrIDToPid:     make(map[string]int),
		containers:     make(map[string]string), // containerName -> containerID
		pidFileHandles: make(map[int]*os.File),
	}, nil
}

func (p *PNSExecutor) GetFileContents(containerName string, sourcePath string) (string, error) {
	err := p.enterChroot(containerName)
	if err != nil {
		return "", err
	}
	defer func() { _ = p.exitChroot() }()
	out, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// enterChroot enters chroot of the main container
func (p *PNSExecutor) enterChroot(containerName string) error {
	pid, err := p.getContainerPID(containerName)
	if err != nil {
		return fmt.Errorf("failed to get container PID: %w", err)
	}
	if err := p.pidFileHandles[pid].Chdir(); err != nil {
		return errors.InternalWrapErrorf(err, "failed to chdir to main filesystem: %v", err)
	}
	if err := osspecific.CallChroot(); err != nil {
		return errors.InternalWrapErrorf(err, "failed to chroot to main filesystem: %v", err)
	}
	return nil
}

// exitChroot exits chroot
func (p *PNSExecutor) exitChroot() error {
	if err := p.rootFS.Chdir(); err != nil {
		return errors.InternalWrapError(err)
	}
	err := osspecific.CallChroot()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

// CopyFile copies a source file in a container to a local path
func (p *PNSExecutor) CopyFile(containerName string, sourcePath string, destPath string, compressionLevel int) (err error) {
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() {
		// exit chroot and close the file. preserve the original error
		deferErr := p.exitChroot()
		if err == nil && deferErr != nil {
			err = errors.InternalWrapError(deferErr)
		}
		deferErr = destFile.Close()
		if err == nil && deferErr != nil {
			err = errors.InternalWrapError(deferErr)
		}
	}()
	w := bufio.NewWriter(destFile)
	err = p.enterChroot(containerName)
	if err != nil {
		return err
	}

	err = archive.TarGzToWriter(sourcePath, compressionLevel, w)
	return err
}

func (p *PNSExecutor) Wait(ctx context.Context, containerNames []string) error {

	go p.pollRootProcesses(containerNames)

	// Secure a filehandle on our own root. This is because we will chroot back and forth from
	// the main container's filesystem, to our own.
	rootFS, err := os.Open("/")
	if err != nil {
		return fmt.Errorf("failed to open my own root: %w", err)
	}
	p.rootFS = rootFS

	if !p.haveContainers(containerNames) { // allow some additional time for polling to get this data
		time.Sleep(3 * time.Second)
	}

	if !p.haveContainers(containerNames) {
		log.Info("container PID still unknown (maybe due to short running main container)")
		err := p.K8sAPIExecutor.Until(ctx, func(pod *corev1.Pod) bool {
			for _, c := range pod.Status.ContainerStatuses {
				containerID := execcommon.GetContainerID(c.ContainerID)
				p.containers[c.Name] = containerID
				log.Infof("mapped container name %q to container ID %q", c.Name, containerID)
			}
			return p.haveContainers(containerNames)
		})
		if err != nil {
			return err
		}
	}

	mainPID, err := p.getContainerPID(common.MainContainerName)
	if err != nil {
		return err
	}

	// wait for pid to complete
	log.Infof("Waiting for main pid %d to complete", mainPID)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			p, err := gops.FindProcess(mainPID)
			if err != nil {
				return fmt.Errorf("failed to find main process: %w", err)
			}
			if p == nil {
				log.Infof("Main pid %d completed", mainPID)
				return nil
			}
			time.Sleep(3 * time.Second)
		}
	}
}

// pollRootProcesses will poll /proc for root pids (pids without parents) in a tight loop, for the
// purpose of securing an open file handle against /proc/<pid>/root as soon as possible.
// It opens file handles on all root pids because at this point, we do not yet know which pid is the
// "main" container.
// Polling is necessary because it is not possible to use something like fsnotify against procfs.
func (p *PNSExecutor) pollRootProcesses(containerNames []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = p.secureRootFiles()
			if p.haveContainers(containerNames) {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (d *PNSExecutor) haveContainers(containerNames []string) bool {
	for _, n := range containerNames {
		if d.containers[n] == "" {
			return false
		}
	}
	return true
}

// Kill a list of containers first with a SIGTERM then with a SIGKILL after a grace period
func (p *PNSExecutor) Kill(ctx context.Context, containerNames []string) error {
	var asyncErr error
	wg := sync.WaitGroup{}
	for _, containerName := range containerNames {
		wg.Add(1)
		go func(containerName string) {
			err := p.killContainer(ctx, containerName)
			if err != nil && asyncErr != nil {
				asyncErr = err
			}
			wg.Done()
		}(containerName)
	}
	wg.Wait()
	return asyncErr
}

func (p *PNSExecutor) killContainer(ctx context.Context, containerName string) error {
	pid, err := p.getContainerPID(containerName)
	if err != nil {
		log.Warnf("Ignoring kill container failure of %q: %v. Process assumed to have completed", containerName, err)
		return nil
	}
	// On Unix systems, FindProcess always succeeds and returns a Process
	// for the given pid, regardless of whether the process exists.
	proc, _ := os.FindProcess(pid)
	log.Infof("Sending SIGTERM to pid %d", pid)
	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		log.Warnf("Failed to SIGTERM pid %d: %v", pid, err)
	}

	waitPIDOpts := executil.WaitPIDOpts{Timeout: execcommon.KillGracePeriod * time.Second}
	err = executil.WaitPID(pid, waitPIDOpts)
	if err == nil {
		log.Infof("PID %d completed", pid)
		return nil
	}
	if err != executil.ErrWaitPIDTimeout {
		return err
	}
	log.Warnf("Timed out (%v) waiting for pid %d to complete after SIGTERM. Issuing SIGKILL", waitPIDOpts.Timeout, pid)
	err = proc.Signal(syscall.SIGKILL)
	if err != nil {
		log.Warnf("Failed to SIGKILL pid %d: %v", pid, err)
	}
	return err
}

// getContainerPID returns the pid associated with the container id. Returns error if it was unable
// to be determined because no running root processes exist with that container ID
func (p *PNSExecutor) getContainerPID(containerName string) (int, error) {
	containerID, ok := p.containers[containerName]
	if !ok {
		return 0, fmt.Errorf("container ID not found for container name %q", containerName)
	}
	pid := p.ctrIDToPid[containerID]
	if pid == 0 {
		return 0, fmt.Errorf("pid not found for container ID %q", containerID)
	}
	return pid, nil
}

func containerNameForPID(pid int) (string, error) {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return "", err
	}
	prefix := common.EnvVarContainerName + "="
	for _, l := range strings.Split(string(data), "\000") {
		if strings.HasPrefix(l, prefix) {
			return strings.TrimPrefix(l, prefix), nil
		}
	}
	return "", errContainerNameNotFound
}

func (p *PNSExecutor) secureRootFiles() error {
	processes, err := gops.Processes()
	if err != nil {
		return err
	}
	for _, proc := range processes {
		_ = func() error {
			pid := proc.Pid()
			if pid == 1 || pid == p.thisPID || proc.PPid() != 0 {
				// ignore the pause container, our own pid, and non-root processes
				return nil
			}

			fs, err := os.Open(fmt.Sprintf("/proc/%d/root", pid))
			if err != nil {
				return err
			}

			// the main container may have switched (e.g. gone from busybox to the user's container)
			if prevInfo, ok := p.pidFileHandles[pid]; ok {
				_ = prevInfo.Close()
			}
			p.pidFileHandles[pid] = fs
			log.Infof("secured root for pid %d root: %s", pid, proc.Executable())

			containerID, err := parseContainerID(pid)
			if err != nil {
				return err
			}
			p.ctrIDToPid[containerID] = pid
			log.Infof("mapped pid %q to container ID %q", pid, containerID)
			containerName, err := containerNameForPID(pid)
			if err != nil {
				return err
			}
			p.containers[containerName] = containerID
			log.Infof("mapped container name %q to container ID %q", containerName, containerID)
			return nil
		}()
	}
	return nil
}

// parseContainerID parses the containerID of a pid
func parseContainerID(pid int) (string, error) {
	cgroupPath := fmt.Sprintf("/proc/%d/cgroup", pid)
	cgroupFile, err := os.OpenFile(cgroupPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	defer func() { _ = cgroupFile.Close() }()
	sc := bufio.NewScanner(cgroupFile)
	for sc.Scan() {
		line := sc.Text()
		log.Debugf("pid %d: %s", pid, line)
		containerID := parseContainerIDFromCgroupLine(line)
		if containerID != "" {
			return containerID, nil
		}
	}
	return "", errors.InternalErrorf("Failed to parse container ID from %s", cgroupPath)
}

func parseContainerIDFromCgroupLine(line string) string {
	// See https://www.systutorials.com/docs/linux/man/5-proc/ for /proc/XX/cgroup format. e.g.:
	// 5:cpuacct,cpu,cpuset:/daemons
	parts := strings.Split(line, "/")
	if len(parts) > 1 {
		if containerID := parts[len(parts)-1]; containerID != "" {
			// need to check for empty string because the line may look like: 5:rdma:/

			// remove possible ".scope" suffix
			containerID := strings.TrimSuffix(containerID, ".scope")

			// for compatibility with cri-containerd record format when using systemd cgroup path
			// example record in /proc/{pid}/cgroup:
			// 9:cpuset:/kubepods-besteffort-pod30556cce_0f92_11eb_b36d_02623cf324c8.slice:cri-containerd:c688c856b21cfb29c1dbf6c14793435e44a1299dfc12add33283239bffed2620
			if strings.Contains(containerID, "cri-containerd") {
				strList := strings.Split(containerID, ":")
				containerID = strList[len(strList)-1]
			}

			// remove possible "*-" prefix
			// e.g. crio-7a92a067289f6197148912be1c15f20f0330c7f3c541473d3b9c4043ca137b42.scope
			parts := strings.Split(containerID, "-")
			containerID = parts[len(parts)-1]

			return containerID
		}
	}
	return ""
}
