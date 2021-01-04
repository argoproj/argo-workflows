package pns

import (
	"bufio"
	"fmt"
	"io"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util/archive"
	errorsutil "github.com/argoproj/argo/util/errors"
	"github.com/argoproj/argo/workflow/common"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
	argowait "github.com/argoproj/argo/workflow/executor/common/wait"
	osspecific "github.com/argoproj/argo/workflow/executor/os-specific"
)

type PNSExecutor struct {
	clientset *kubernetes.Clientset
	podName   string
	namespace string

	// ctrIDToPid maps a containerID to a process ID
	ctrIDToPid map[string]int
	// pidToCtrID maps a process ID to a container ID
	pidToCtrID map[int]string

	// pidFileHandles holds file handles to all root containers
	pidFileHandles map[int]*fileInfo

	// thisPID is the pid of this process
	thisPID int
	// mainFS holds a file descriptor to the main filesystem, allowing the executor to access the
	// filesystem after the main process exited
	mainFS *os.File
	// rootFS holds a file descriptor to the root filesystem, allowing the executor to exit out of a chroot
	rootFS *os.File
	// debug enables additional debugging
	debug bool
	// hasOutputs indicates if the template has outputs. determines if we need to
	hasOutputs bool
}

type fileInfo struct {
	file os.File
	info os.FileInfo
}

func NewPNSExecutor(clientset *kubernetes.Clientset, podName, namespace string, hasOutputs bool) (*PNSExecutor, error) {
	thisPID := os.Getpid()
	log.Infof("Creating PNS executor (namespace: %s, pod: %s, pid: %d, hasOutputs: %v)", namespace, podName, thisPID, hasOutputs)
	if thisPID == 1 {
		return nil, errors.New(errors.CodeBadRequest, "process namespace sharing is not enabled on pod")
	}
	return &PNSExecutor{
		clientset:      clientset,
		podName:        podName,
		namespace:      namespace,
		ctrIDToPid:     make(map[string]int),
		pidToCtrID:     make(map[int]string),
		pidFileHandles: make(map[int]*fileInfo),
		thisPID:        thisPID,
		debug:          log.GetLevel() == log.DebugLevel,
		hasOutputs:     hasOutputs,
	}, nil
}

func (p *PNSExecutor) GetFileContents(containerID string, sourcePath string) (string, error) {
	err := p.enterChroot()
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
func (p *PNSExecutor) enterChroot() error {
	if p.mainFS == nil {
		return errors.InternalErrorf("could not chroot into main for artifact collection: container may have exited too quickly")
	}
	if err := p.mainFS.Chdir(); err != nil {
		return errors.InternalWrapErrorf(err, "failed to chdir to main filesystem: %v", err)
	}
	err := osspecific.CallChroot()
	if err != nil {
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
func (p *PNSExecutor) CopyFile(containerID string, sourcePath string, destPath string, compressionLevel int) (err error) {
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
	err = p.enterChroot()
	if err != nil {
		return err
	}

	err = archive.TarGzToWriter(sourcePath, compressionLevel, w)
	return err
}

func (p *PNSExecutor) WaitInit() error {
	if !p.hasOutputs {
		return nil
	}
	go p.pollRootProcesses(time.Minute)
	// Secure a filehandle on our own root. This is because we will chroot back and forth from
	// the main container's filesystem, to our own.
	rootFS, err := os.Open("/")
	if err != nil {
		return errors.InternalWrapError(err)
	}
	p.rootFS = rootFS
	return nil
}

// Wait for the container to complete
func (p *PNSExecutor) Wait(containerID string) error {
	mainPID, err := p.getContainerPID(containerID)
	if err != nil {
		log.Warnf("Failed to get main PID: %v", err)
		if !p.hasOutputs {
			log.Warnf("Ignoring wait failure: %v. Process assumed to have completed", err)
			return nil
		}
		return argowait.UntilTerminated(p.clientset, p.namespace, p.podName, containerID)
	}
	log.Infof("Main pid identified as %d", mainPID)
	for pid, f := range p.pidFileHandles {
		if pid == mainPID {
			log.Info("Successfully secured file handle on main container root filesystem")
			p.mainFS = &f.file
		} else {
			log.Infof("Closing root filehandle for non-main pid %d", pid)
			_ = f.file.Close()
		}
	}
	if p.mainFS == nil {
		log.Warn("Failed to secure file handle on main container's root filesystem. Output artifacts from base image layer will fail")
	}

	// wait for pid to complete
	log.Infof("Waiting for main pid %d to complete", mainPID)
	err = executil.WaitPID(mainPID)
	if err != nil {
		return err
	}
	log.Infof("Main pid %d completed", mainPID)
	return nil
}

// pollRootProcesses will poll /proc for root pids (pids without parents) in a tight loop, for the
// purpose of securing an open file handle against /proc/<pid>/root as soon as possible.
// It opens file handles on all root pids because at this point, we do not yet know which pid is the
// "main" container.
// Polling is necessary because it is not possible to use something like fsnotify against procfs.
func (p *PNSExecutor) pollRootProcesses(timeout time.Duration) {
	log.Warnf("Polling root processes (%v)", timeout)
	deadline := time.Now().Add(timeout)
	for {
		p.updateCtrIDMap()
		if p.mainFS != nil {
			log.Info("Stopped root processes polling due to successful securing of main root fs")
			break
		}
		if time.Now().After(deadline) {
			log.Warnf("Polling root processes timed out (%v)", timeout)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (p *PNSExecutor) GetOutputStream(containerID string, combinedOutput bool) (io.ReadCloser, error) {
	if !combinedOutput {
		log.Warn("non combined output unsupported")
	}
	opts := corev1.PodLogOptions{
		Container: common.MainContainerName,
		Follow:    true,
	}
	return p.clientset.CoreV1().Pods(p.namespace).GetLogs(p.podName, &opts).Stream()
}

func (p *PNSExecutor) GetExitCode(containerID string) (string, error) {
	log.Infof("Getting exit code of %s", containerID)
	_, containerStatus, err := p.GetTerminatedContainerStatus(containerID)
	if err != nil {
		return "", fmt.Errorf("could not get container status: %s", err)
	}
	if containerStatus.State.Terminated != nil {
		return fmt.Sprint(containerStatus.State.Terminated.ExitCode), nil
	}
	return "", nil
}

// Kill a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (p *PNSExecutor) Kill(containerIDs []string) error {
	var asyncErr error
	wg := sync.WaitGroup{}
	for _, cid := range containerIDs {
		wg.Add(1)
		go func(containerID string) {
			err := p.killContainer(containerID)
			if err != nil && asyncErr != nil {
				asyncErr = err
			}
			wg.Done()
		}(cid)
	}
	wg.Wait()
	return asyncErr
}

func (p *PNSExecutor) killContainer(containerID string) error {
	pid, err := p.getContainerPID(containerID)
	if err != nil {
		log.Warnf("Ignoring kill container failure of %s: %v. Process assumed to have completed", containerID, err)
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
func (p *PNSExecutor) getContainerPID(containerID string) (int, error) {
	pid, ok := p.ctrIDToPid[containerID]
	if ok {
		return pid, nil
	}
	p.updateCtrIDMap()
	pid, ok = p.ctrIDToPid[containerID]
	if !ok {
		return -1, errors.InternalErrorf("Failed to determine pid for containerID %s: container may have exited too quickly", containerID)
	}
	return pid, nil
}

// updateCtrIDMap updates the mapping between container IDs to PIDs
func (p *PNSExecutor) updateCtrIDMap() {
	allProcs, err := gops.Processes()
	if err != nil {
		log.Warnf("Failed to list processes: %v", err)
		return
	}
	for _, proc := range allProcs {
		pid := proc.Pid()
		if pid == 1 || pid == p.thisPID || proc.PPid() != 0 {
			// ignore the pause container, our own pid, and non-root processes
			continue
		}

		// Useful code for debugging:
		if p.debug {
			if data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/root", pid) + "/etc/os-release"); err == nil {
				log.Infof("pid %d: %s", pid, string(data))
				_, _ = parseContainerID(pid)
			}
		}

		if p.hasOutputs && p.mainFS == nil {
			rootPath := fmt.Sprintf("/proc/%d/root", pid)
			currInfo, err := os.Stat(rootPath)
			if err != nil {
				log.Warnf("Failed to stat %s: %v", rootPath, err)
				continue
			}
			log.Infof("pid %d: %v", pid, currInfo)
			prevInfo := p.pidFileHandles[pid]

			// Secure the root filehandle of the process. NOTE if the file changed, it means that
			// the main container may have switched (e.g. gone from busybox to the user's container)
			if prevInfo == nil || !os.SameFile(prevInfo.info, currInfo) {
				fs, err := os.Open(rootPath)
				if err != nil {
					log.Warnf("Failed to open %s: %v", rootPath, err)
					continue
				}
				log.Infof("Secured filehandle on %s", rootPath)
				p.pidFileHandles[pid] = &fileInfo{
					info: currInfo,
					file: *fs,
				}
				if prevInfo != nil {
					_ = prevInfo.file.Close()
				}
			}
		}

		// Update maps of pids to container ids
		if _, ok := p.pidToCtrID[pid]; !ok {
			containerID, err := parseContainerID(pid)
			if err != nil {
				log.Warnf("Failed to identify containerID for process %d", pid)
				continue
			}
			log.Infof("containerID %s mapped to pid %d", containerID, pid)
			p.ctrIDToPid[containerID] = pid
			p.pidToCtrID[pid] = containerID
		}
	}
}

var backoffOver30s = wait.Backoff{
	Duration: 1 * time.Second,
	Steps:    7,
	Factor:   2,
}

func (p *PNSExecutor) GetTerminatedContainerStatus(containerID string) (*corev1.Pod, *corev1.ContainerStatus, error) {
	var pod *corev1.Pod
	var containerStatus *corev1.ContainerStatus
	// Under high load, the Kubernetes API may be unresponsive for some time (30s). This would have failed the workflow
	// previously (<=v2.11) but a 30s back-off mitigates this.
	err := wait.ExponentialBackoff(backoffOver30s, func() (bool, error) {
		podRes, err := p.clientset.CoreV1().Pods(p.namespace).Get(p.podName, metav1.GetOptions{})
		if err != nil {
			return !errorsutil.IsTransientErr(err), fmt.Errorf("could not get pod: %w", err)
		}
		for _, containerStatusRes := range podRes.Status.ContainerStatuses {
			if execcommon.GetContainerID(&containerStatusRes) != containerID {
				continue
			}
			pod = podRes
			containerStatus = &containerStatusRes
			return containerStatus.State.Terminated != nil, nil
		}
		return false, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod %s", containerID, p.podName))
	})
	return pod, containerStatus, err
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
