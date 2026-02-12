package osspecific

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

var (
	Term = os.Interrupt

	modkernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procCreateRemoteThread = modkernel32.NewProc("CreateRemoteThread")
	procCtrlRoutine        = modkernel32.NewProc("CtrlRoutine")
)

func CanIgnoreSignal(s os.Signal) bool {
	return false
}

func Kill(pid int, s syscall.Signal) error {
	if pid < 0 {
		pid = -pid // // we cannot kill a negative process on windows
	}

	winSignal := -1
	switch s {
	case syscall.SIGTERM:
		winSignal = windows.CTRL_SHUTDOWN_EVENT
	case syscall.SIGINT:
		winSignal = windows.CTRL_C_EVENT
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
