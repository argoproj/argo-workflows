package pns

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	executil "github.com/argoproj/pkg/exec"
	gops "github.com/mitchellh/go-ps"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/util/archive"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/k8sapi"
	osspecific "github.com/argoproj/argo-workflows/v3/workflow/executor/os-specific"
)

const anonymousPIDPrefix = "pid/"

type PNSExecutor struct {
	*k8sapi.K8sAPIExecutor
	podName   string
	namespace string

	// mu for `containerNameToPID``
	mu sync.RWMutex

	containerNameToPID map[string]int

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
	delegate := k8sapi.NewK8sAPIExecutor(clientset, nil, podName, namespace)
	return &PNSExecutor{
		K8sAPIExecutor:     delegate,
		podName:            podName,
		namespace:          namespace,
		mu:                 sync.RWMutex{},
		containerNameToPID: make(map[string]int),
		pidFileHandles:     make(map[int]*os.File),
		thisPID:            thisPID,
	}, nil
}

func (p *PNSExecutor) GetFileContents(containerName string, sourcePath string) (string, error) {
	err := p.enterChroot(containerName)
	if err != nil {
		return "", err
	}
	defer func() { _ = p.exitChroot() }()
	out, err := ioutil.ReadFile(filepath.Clean(sourcePath))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// enterChroot enters chroot of the main container
func (p *PNSExecutor) enterChroot(containerName string) error {
	pid := p.getContainerPID(containerName)
	if pid == 0 {
		return fmt.Errorf("cannot enter chroot for container named %q: no PID known - maybe short running container", containerName)
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
	go p.pollRootProcesses(ctx, containerNames)

	// Secure a filehandle on our own root. This is because we will chroot back and forth from
	// the main container's filesystem, to our own.
	rootFS, err := os.Open("/")
	if err != nil {
		return fmt.Errorf("failed to open my own root: %w", err)
	}
	p.rootFS = rootFS

	/*
		What is a "short running container" and "late starting container"?:

		Short answer: any container that exits in <5s

		Long answer:

		Some containers are short running and we cannot determine their PIDs because they exit too quickly.
		This loop allows 5s for `pollRootProcesses` find PIDs, so we define any container that exits <5s as short running

		Unfortunately, we cannot assume that a container that did not appeared within the 5s has completed.
		They may still be in `ContainerCreating` state - i.e. late starting.
	*/
	for i := 0; !p.haveContainerPIDs(containerNames) && i < 5; i++ {
		time.Sleep(1 * time.Second)
	}

	return p.K8sAPIExecutor.Wait(ctx, containerNames)
}

// pollRootProcesses will poll /proc for root pids (pids without parents) in a tight loop, for the
// purpose of securing an open file handle against /proc/<pid>/root as soon as possible.
// It opens file handles on all root pids because at this point, we do not yet know which pid is the
// "main" container.
// Polling is necessary because it is not possible to use something like fsnotify against procfs.
func (p *PNSExecutor) pollRootProcesses(ctx context.Context, containerNames []string) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := p.secureRootFiles(); err != nil {
				log.WithError(err).Warn("failed to secure root files")
			}
			if p.haveContainerPIDs(containerNames) {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (p *PNSExecutor) haveContainerPIDs(containerNames []string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, n := range containerNames {
		if p.containerNameToPID[n] == 0 {
			return false
		}
	}
	return true
}

// Kill a list of containers first with a SIGTERM then with a SIGKILL after a grace period
func (p *PNSExecutor) Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	var asyncErr error
	wg := sync.WaitGroup{}
	for _, containerName := range containerNames {
		wg.Add(1)
		go func(containerName string) {
			err := p.killContainer(containerName, terminationGracePeriodDuration)
			if err != nil && asyncErr != nil {
				asyncErr = err
			}
			wg.Done()
		}(containerName)
	}
	wg.Wait()
	return asyncErr
}

func (p *PNSExecutor) ListContainerNames(ctx context.Context) ([]string, error) {
	procs, err := gops.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}
	var containerNames []string
	for _, proc := range procs {
		if !p.isRootContainerProcess(proc) {
			continue
		}
		n, err := containerNameForPID(proc.Pid())
		if err != nil {
			return nil, fmt.Errorf("failed to get container name for process %d: %w", proc.Pid(), err)
		}
		containerNames = append(containerNames, n)
	}
	return containerNames, nil
}

func (p *PNSExecutor) killContainer(containerName string, terminationGracePeriodDuration time.Duration) error {
	pid := p.getContainerPID(containerName)
	if pid == 0 {
		log.Warnf("No PID for container named %q. Process assumed to have completed", containerName)
		return nil
	}
	// On Unix systems, FindProcess always succeeds and returns a Process
	// for the given pid, regardless of whether the process exists.
	proc, _ := os.FindProcess(pid)
	log.Infof("Sending SIGTERM to pid %d", pid)
	err := proc.Signal(syscall.SIGTERM)
	if err != nil {
		log.Warnf("Failed to SIGTERM pid %d: %v", pid, err)
	}
	waitPIDOpts := executil.WaitPIDOpts{Timeout: terminationGracePeriodDuration}
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

// returns the entries associated with the container id. Returns zero if it was unable
// to be determined because no running root processes exist with that container ID
func (p *PNSExecutor) getContainerPID(containerName string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if pid, ok := p.containerNameToPID[containerName]; ok {
		return pid
	}
	procs, _ := gops.Processes()
	for _, proc := range procs {
		n, _ := containerNameForPID(proc.Pid())
		if n == containerName {
			return proc.Pid()
		}
	}
	for n, pid := range p.containerNameToPID {
		// the container can't be me, and it must be anonymous, otherwise we would have determined it
		if pid != os.Getpid() && strings.HasPrefix(n, anonymousPIDPrefix) {
			log.Infof("guessing container %q PID is %d", containerName, pid)
			return pid
		}
	}
	return 0
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
	return fmt.Sprintf("%s%d", anonymousPIDPrefix, pid), nil // we give all a "container name", including a fake name for injected sidecars
}

func (p *PNSExecutor) secureRootFiles() error {
	processes, err := gops.Processes()
	if err != nil {
		return err
	}
	for _, proc := range processes {
		err = func() error {
			pid := proc.Pid()
			if !p.isRootContainerProcess(proc) {
				return nil
			}

			fs, err := os.Open(fmt.Sprintf("/proc/%d/root", pid))
			if err != nil {
				return err
			}

			if p.pidFileHandles[pid] != fs {
				// the main container may have switched (e.g. gone from busybox to the user's container)
				if prevInfo, ok := p.pidFileHandles[pid]; ok {
					_ = prevInfo.Close()
				}
				p.pidFileHandles[pid] = fs
				log.Infof("secured root for pid %d root: %s (%q)", pid, proc.Executable(), fs.Name())
			}

			containerName, err := containerNameForPID(pid)
			if err != nil {
				return err
			}
			if p.containerNameToPID[containerName] != pid {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.containerNameToPID[containerName] = pid
				log.Infof("mapped container name %q to pid %d", containerName, pid)
			}
			return nil
		}()
		if err != nil {
			log.WithError(err).Warnf("failed to secure root file handle for %d", proc.Pid())
		}
	}
	return nil
}

func (p *PNSExecutor) isRootContainerProcess(proc gops.Process) bool {
	// ignore the pause container, our own pid, and non-root processes
	return proc.Pid() != 1 && proc.Pid() != p.thisPID && proc.PPid() == 0
}
