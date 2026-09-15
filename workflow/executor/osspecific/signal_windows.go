package osspecific

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/argoproj/argo-workflows/v4/util/errors"
)

var (
	Term = os.Interrupt

	modkernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procCreateRemoteThread       = modkernel32.NewProc("CreateRemoteThread")
	procCtrlRoutine              = modkernel32.NewProc("CtrlRoutine")
	procCreateToolhelp32Snapshot = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modkernel32.NewProc("Process32FirstW")
	procProcess32Next            = modkernel32.NewProc("Process32NextW")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260
)

type ProcessEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [MAX_PATH]uint16
}

func CanIgnoreSignal(s os.Signal) bool {
	return false
}

func createToolhelp32Snapshot(flags uint32, processID uint32) (handle windows.Handle, err error) {
	r0, _, e1 := syscall.SyscallN(procCreateToolhelp32Snapshot.Addr(), uintptr(flags), uintptr(processID))
	handle = windows.Handle(r0)
	if handle == 0 {
		err = errnoErr(e1)
	}
	return
}

func process32First(snapshot windows.Handle, procEntry *ProcessEntry32) (err error) {
	r0, _, e1 := syscall.SyscallN(procProcess32First.Addr(), uintptr(snapshot), uintptr(unsafe.Pointer(procEntry)))
	if r0 == 0 {
		err = errnoErr(e1)
	}
	return
}

func process32Next(snapshot windows.Handle, procEntry *ProcessEntry32) (err error) {
	r0, _, e1 := syscall.SyscallN(procProcess32Next.Addr(), uintptr(snapshot), uintptr(unsafe.Pointer(procEntry)))
	if r0 == 0 {
		err = errnoErr(e1)
	}
	return
}

func findProcessByName(name string) (int, error) {
	snapshot, err := createToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to create process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var pe ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	err = process32First(snapshot, &pe)
	if err != nil {
		return -1, fmt.Errorf("failed to get first process: %w", err)
	}

	nameLower := strings.ToLower(name)

	for {
		exeFile := windows.UTF16ToString(pe.ExeFile[:])
		exeFileLower := strings.ToLower(exeFile)

		if exeFileLower == nameLower || exeFileLower == nameLower+".exe" {
			return int(pe.ProcessID), nil
		}

		err = process32Next(snapshot, &pe)
		if err != nil {
			break
		}
	}

	return -1, fmt.Errorf("process %s not found", name)
}

func Kill(pid int, s syscall.Signal) error {
	// Special case: if pid is 1, find the process named "argoexec"
	if pid == 1 {
		execPid, err := findProcessByName("argoexec")
		if err != nil {
			return fmt.Errorf("failed to find argoexec process: %w", err)
		}
		pid = execPid
	} else if pid < 0 {
		pid = -pid // we cannot kill a negative process on windows
	}

	winSignal := -1
	switch s {
	case syscall.SIGTERM:
		winSignal = windows.CTRL_SHUTDOWN_EVENT
	case syscall.SIGINT:
		winSignal = windows.CTRL_C_EVENT
	case syscall.SIGKILL:
		winSignal = windows.CTRL_SHUTDOWN_EVENT
	}

	if winSignal == -1 {
		p, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return p.Signal(s)
	}
	return signalProcess(uint32(pid), winSignal)
}

func Setpgid(a *syscall.SysProcAttr) {
	// this does not exist on windows
}

func Wait(process *os.Process) error {
	stat, err := process.Wait()
	if stat.ExitCode() != 0 {
		return errors.NewExitErr(stat.ExitCode())
	}
	return err
}

// signalProcess sends the specified signal to a process.
//
// Code +/- copied from: https://github.com/microsoft/hcsshim/blob/1d69a9c658655b77dd4e5275bff99caad6b38416/internal/jobcontainers/process.go#L251
// License: MIT
// Author: Microsoft
func signalProcess(pid uint32, signal int) error {
	hProc, err := windows.OpenProcess(windows.PROCESS_TERMINATE, true, pid)
	if err != nil {
		return fmt.Errorf("failed to open process: %w", err)
	}
	defer func() {
		_ = windows.Close(hProc)
	}()

	if err := procCtrlRoutine.Find(); err != nil {
		return fmt.Errorf("failed to load CtrlRoutine: %w", err)
	}

	threadHandle, err := createRemoteThread(hProc, nil, 0, procCtrlRoutine.Addr(), uintptr(signal), 0, nil)
	if err != nil {
		return fmt.Errorf("failed to open remote thread in target process %d: %w", pid, err)
	}
	defer func() {
		_ = windows.Close(windows.Handle(threadHandle))
	}()
	return nil
}

// Following code has been generated using github.com/Microsoft/go-winio/tools/mkwinsyscall and inlined
// for easier usage

// HANDLE CreateRemoteThread(
//
//	HANDLE                 hProcess,
//	LPSECURITY_ATTRIBUTES  lpThreadAttributes,
//	SIZE_T                 dwStackSize,
//	LPTHREAD_START_ROUTINE lpStartAddress,
//	LPVOID                 lpParameter,
//	DWORD                  dwCreationFlags,
//	LPDWORD                lpThreadId
//
// );
func createRemoteThread(process windows.Handle, sa *windows.SecurityAttributes, stackSize uint32, startAddr uintptr, parameter uintptr, creationFlags uint32, threadID *uint32) (handle windows.Handle, err error) {
	r0, _, e1 := syscall.SyscallN(procCreateRemoteThread.Addr(), uintptr(process), uintptr(unsafe.Pointer(sa)), uintptr(stackSize), uintptr(startAddr), uintptr(parameter), uintptr(creationFlags), uintptr(unsafe.Pointer(threadID)))
	handle = windows.Handle(r0)
	if handle == 0 {
		err = errnoErr(e1)
	}
	return
}

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	return e
}
