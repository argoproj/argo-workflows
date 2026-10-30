// Package poolpod is a pool pod's side of hot workers: the file protocol
// between its `worker` and `main` containers, and what `main` needs to run
// one task after another in the same container without them seeing each
// other: the process sweep and the isolation of each task's files.
//
// The protocol works over the shared /var/run/argo volume. The pool-level
// directory Dir holds `current`, the path of the task directory `main`
// should run next, and `main`, main's liveness lock, held exclusively for
// as long as main is taking tasks, which the kernel releases however main
// dies: the worker claims a task only while main holds it, so a main that
// can't start (its image can't be pulled, or its env names a missing
// Secret) leaves the pool's tasks queued. Each task directory holds:
//
//   - `task`: the task's TaskSpec, written by the worker before it offers
//     the task.
//   - `ready`: written by the worker once the task directory is complete,
//     just before the worker points `current` at it.
//   - `taken`: created exclusively, so whoever creates it first wins: by
//     main with the content `main` when it starts the task, or by the worker
//     with `withdrawn` to stop main ever starting it.
//   - `lock`: main's liveness lock, held exclusively while main has the task,
//     which the kernel releases however main dies.
//   - `message`: optional, main's message for a task it failed itself (such
//     as an input path that exists in the image), written before `exitcode`.
//   - `exitcode`: the task's exit code, written by main before it releases
//     `lock`.
//   - `stop`: written by the worker to make main stop the task, holding the
//     reason.
//
// Where task directories live is not this package's business: the
// executor's per-task path layout decides (in a pool pod, a directory per
// task in the pool directory), and both sides name a task by its directory.
package poolpod

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// Dir is the pool-level directory on the shared volume.
const Dir = common.VarRunArgoPath + "/" + DirName

// The protocol's file names: CurrentFile in the pool directory, the rest in
// each task directory.
const (
	CurrentFile  = "current"
	MainFile     = "main"
	ReadyFile    = "ready"
	TakenFile    = "taken"
	LockFile     = "lock"
	ExitCodeFile = "exitcode"
	MessageFile  = "message"
	StopFile     = "stop"
)

// The content of `taken`: who created it.
const (
	takenByMain = "main"
	withdrawn   = "withdrawn"
)

// StopReason is the content of `stop`: why the worker stopped the task.
type StopReason string

const (
	// StopDeadline: the task's deadline passed.
	StopDeadline StopReason = "deadline"
	// StopInterrupt: the controller ended the node and labelled the result
	// interrupted.
	StopInterrupt StopReason = "interrupt"
	// StopLost: the worker lost its claim on the task.
	StopLost StopReason = "lost"
	// StopRelease: the worker is shutting down and gives the task back.
	StopRelease StopReason = "release"
)

func (r StopReason) valid() bool {
	switch r {
	case StopDeadline, StopInterrupt, StopLost, StopRelease:
		return true
	}
	return false
}

var (
	// ErrNotOffered is returned by Take for a task directory without
	// `ready`, including one the worker has already deleted.
	ErrNotOffered = errors.New("task not offered")
	// ErrWithdrawn is returned by Take when the worker withdrew the task
	// first.
	ErrWithdrawn = errors.New("task withdrawn")
	// ErrTaken is returned by Take when main already took the task: a
	// restarted main never re-runs a task its predecessor took.
	ErrTaken = errors.New("task already taken")
)

// lockRetryInterval is how often Take retries the lock while someone else
// holds it. Only the worker's momentary LockHeld probes contend with main.
const lockRetryInterval = 10 * time.Millisecond

// Pool is the pool-level directory, holding `current`.
type Pool struct {
	dir string
}

// NewPool returns the pool whose directory is dir; production uses Dir.
func NewPool(dir string) Pool {
	return Pool{dir: dir}
}

// Dir is the pool's directory.
func (p Pool) Dir() string {
	return p.dir
}

// Init creates the pool directory, writable by both containers, which may
// run as different users. Either side may call it, in either order.
func (p Pool) Init() error {
	return MakeSharedDir(p.dir)
}

// mainUpInterval is how often WaitMainUp probes main's liveness lock.
const mainUpInterval = 200 * time.Millisecond

// HoldMain takes main's liveness lock, waiting while a worker probes it, and
// holds it until the returned file is closed or main exits. Called by main
// once it is ready to take tasks.
func (p Pool) HoldMain(ctx context.Context) (*os.File, error) {
	return acquire(ctx, filepath.Join(p.dir, MainFile))
}

// MainUp reports whether main holds its liveness lock: whether it is up and
// taking tasks.
func (p Pool) MainUp() (bool, error) {
	return osspecific.LockHeld(filepath.Join(p.dir, MainFile))
}

// WaitMainUp waits until main is up, or ctx ends.
func (p Pool) WaitMainUp(ctx context.Context) error {
	return wait.PollUntilContextCancel(ctx, mainUpInterval, true, func(context.Context) (bool, error) {
		return p.MainUp()
	})
}

// Offer makes the task the one main runs next. Called by the worker once
// the task directory is complete: it writes `ready`, then points `current`
// at the task directory, replacing it atomically.
func (p Pool) Offer(t Task) error {
	if err := writeAtomic(t.dir, ReadyFile, nil); err != nil {
		return fmt.Errorf("mark task %s ready: %w", t.dir, err)
	}
	if err := writeAtomic(p.dir, CurrentFile, []byte(t.dir)); err != nil {
		return fmt.Errorf("publish task %s: %w", t.dir, err)
	}
	return nil
}

// Current returns the task `current` names, and false if no task has been
// offered yet.
func (p Pool) Current() (Task, bool, error) {
	data, err := os.ReadFile(filepath.Join(p.dir, CurrentFile))
	if errors.Is(err, fs.ErrNotExist) {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, err
	}
	if len(data) == 0 {
		return Task{}, false, fmt.Errorf("%s is empty", filepath.Join(p.dir, CurrentFile))
	}
	return NewTask(string(data)), true, nil
}

// WaitNext blocks until `current` names a task other than prev, and returns
// it. Called by main between tasks; pass the zero Task to return whatever
// `current` names, as a newly started main does.
func (p Pool) WaitNext(ctx context.Context, prev Task) (Task, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		next    Task
		readErr error
	)
	err := file.WatchFile(ctx, filepath.Join(p.dir, CurrentFile), func() {
		t, ok, err := p.Current()
		switch {
		case err != nil:
			readErr = err
			cancel()
		case ok && t != prev:
			next = t
			cancel()
		}
	})
	if readErr != nil {
		return Task{}, readErr
	}
	if next != (Task{}) {
		return next, nil
	}
	return Task{}, err
}

// Task is one task's directory.
type Task struct {
	dir string
}

// NewTask returns the task whose directory is dir.
func NewTask(dir string) Task {
	return Task{dir: dir}
}

// Dir is the task's directory.
func (t Task) Dir() string {
	return t.dir
}

// Create creates the task directory, writable by main. Called by the worker
// before it writes the task's files.
func (t Task) Create() error {
	return MakeSharedDir(t.dir)
}

// Withdraw stops main ever starting the task, and reports whether it did:
// false means main has already taken it. Called by the worker; withdrawing
// a task again reports true.
func (t Task) Withdraw() (bool, error) {
	created, err := createExclusive(t.dir, TakenFile, []byte(withdrawn))
	if err != nil || created {
		return created, err
	}
	by, err := t.takenBy()
	if err != nil {
		return false, err
	}
	return by == withdrawn, nil
}

// Stop asks main to stop the task, and returns the reason that stands: the
// first reason written wins, so a later Stop returns the earlier reason.
// Called by the worker; it may be called before main has taken the task.
func (t Task) Stop(reason StopReason) (StopReason, error) {
	if !reason.valid() {
		return "", fmt.Errorf("invalid stop reason %q", reason)
	}
	if _, err := createExclusive(t.dir, StopFile, []byte(reason)); err != nil {
		return "", err
	}
	got, _, err := t.stopReason()
	return got, err
}

// State is where a task is in the protocol.
type State int

const (
	// Offered: main hasn't taken the task, nor the worker withdrawn it.
	Offered State = iota
	// Withdrawn: the worker withdrew the task before main took it.
	Withdrawn
	// Running: main has the task.
	Running
	// Exited: main finished the task and wrote its exit code.
	Exited
	// Died: main took the task and died before writing its exit code. The
	// task is Failed.
	Died
)

func (s State) String() string {
	switch s {
	case Offered:
		return "Offered"
	case Withdrawn:
		return "Withdrawn"
	case Running:
		return "Running"
	case Exited:
		return "Exited"
	case Died:
		return "Died"
	}
	return "State(" + strconv.Itoa(int(s)) + ")"
}

// Status is a task's State and, once Exited, its exit code.
type Status struct {
	State    State
	ExitCode int
	// Message is main's message for an Exited task it failed itself, or
	// empty.
	Message string
}

// Status reads the task's state. Called by the worker. Offered and Running
// may be stale as soon as they are returned: an Offered task can become
// Running or Withdrawn, and a Running one Exited or Died. Withdrawn, Exited
// and Died are final.
func (t Task) Status() (Status, error) {
	by, err := t.takenBy()
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Status{State: Offered}, nil
	case err != nil:
		return Status{}, err
	case by == withdrawn:
		return Status{State: Withdrawn}, nil
	case by != takenByMain:
		return Status{}, fmt.Errorf("%s holds %q", filepath.Join(t.dir, TakenFile), by)
	}
	// main writes exitcode before it releases the lock, so look at the lock
	// first: once it is free, exitcode is either there or never will be.
	held, err := osspecific.LockHeld(filepath.Join(t.dir, LockFile))
	if err != nil {
		return Status{}, err
	}
	if held {
		return Status{State: Running}, nil
	}
	data, err := os.ReadFile(filepath.Join(t.dir, ExitCodeFile))
	if errors.Is(err, fs.ErrNotExist) {
		return Status{State: Died}, nil
	}
	if err != nil {
		return Status{}, err
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return Status{}, fmt.Errorf("parse %s: %w", filepath.Join(t.dir, ExitCodeFile), err)
	}
	// main writes message, if any, before exitcode.
	message, err := os.ReadFile(filepath.Join(t.dir, MessageFile))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Status{}, err
	}
	return Status{State: Exited, ExitCode: code, Message: string(message)}, nil
}

// WaitMain blocks until the task is neither Offered nor Running, and
// returns its status: Withdrawn, Exited or Died. Called by the worker.
func (t Task) WaitMain(ctx context.Context) (Status, error) {
	for {
		s, err := t.Status()
		if err != nil {
			return Status{}, err
		}
		switch s.State {
		case Offered:
			err = file.WaitForCreate(ctx, filepath.Join(t.dir, TakenFile))
		case Running:
			// A restarted main briefly takes the lock of a task its
			// predecessor died in before it skips it, so this can wake
			// to find the lock held again; the loop rechecks.
			err = osspecific.WaitForSharedLock(ctx, filepath.Join(t.dir, LockFile))
		default:
			return s, nil
		}
		if err != nil {
			return Status{}, err
		}
	}
}

// Take is main starting the task: it takes the task's liveness lock, then
// creates `taken` with the content `main`. ErrNotOffered, ErrWithdrawn and
// ErrTaken mean main must skip the task; the lock is released. A task
// directory the worker removes as main takes it is ErrNotOffered too: the
// worker removes a task's directory once it is done with it, while a newly
// started main may be taking the task `current` still names. Called by
// main.
func (t Task) Take(ctx context.Context) (*Run, error) {
	notOffered := func(err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%s: %w", t.dir, ErrNotOffered)
		}
		return err
	}
	if _, err := os.Stat(filepath.Join(t.dir, ReadyFile)); err != nil {
		return nil, notOffered(err)
	}
	lock, err := acquire(ctx, filepath.Join(t.dir, LockFile))
	if err != nil {
		return nil, notOffered(err)
	}
	created, err := createExclusive(t.dir, TakenFile, []byte(takenByMain))
	err = notOffered(err)
	if err == nil && !created {
		var by string
		by, err = t.takenBy()
		err = notOffered(err)
		switch {
		case err != nil:
		case by == withdrawn:
			err = fmt.Errorf("%s: %w", t.dir, ErrWithdrawn)
		default:
			err = fmt.Errorf("%s: %w", t.dir, ErrTaken)
		}
	}
	if err != nil {
		_ = lock.Close()
		return nil, err
	}
	return &Run{task: t, lock: lock}, nil
}

func (t Task) takenBy() (string, error) {
	data, err := os.ReadFile(filepath.Join(t.dir, TakenFile))
	return string(data), err
}

func (t Task) stopReason() (StopReason, bool, error) {
	data, err := os.ReadFile(filepath.Join(t.dir, StopFile))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	reason := StopReason(data)
	if !reason.valid() {
		return "", false, fmt.Errorf("%s holds %q", filepath.Join(t.dir, StopFile), reason)
	}
	return reason, true, nil
}

// Run is main's hold on a task it took.
type Run struct {
	task Task
	lock *os.File
}

// Task is the task being run.
func (r *Run) Task() Task {
	return r.task
}

// StopRequested reports whether the worker has asked main to stop the task,
// and why.
func (r *Run) StopRequested() (StopReason, bool, error) {
	return r.task.stopReason()
}

// WaitStop blocks until the worker asks main to stop the task, and returns
// why.
func (r *Run) WaitStop(ctx context.Context) (StopReason, error) {
	if err := file.WaitForCreate(ctx, filepath.Join(r.task.dir, StopFile)); err != nil {
		return "", err
	}
	reason, _, err := r.task.stopReason()
	return reason, err
}

// Finish writes message, if any, and the task's exit code, then releases the
// lock. message is main's own reason for failing the task, such as an input
// path that exists in the image; it is empty for a task that ran. The lock is
// released even if a write fails, which the worker sees as main having died.
func (r *Run) Finish(exitCode int, message string) error {
	var err error
	if message != "" {
		err = writeAtomic(r.task.dir, MessageFile, []byte(message))
	}
	if err == nil {
		err = writeAtomic(r.task.dir, ExitCodeFile, []byte(strconv.Itoa(exitCode)))
	}
	if closeErr := r.lock.Close(); err == nil {
		err = closeErr
	}
	return err
}

// acquire takes the exclusive lock at path, waiting while someone else
// holds it.
func acquire(ctx context.Context, path string) (*os.File, error) {
	for {
		f, err := osspecific.Acquire(path)
		if err == nil {
			return f, nil
		}
		if !osspecific.IsLockHeld(err) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockRetryInterval):
		}
	}
}

// MakeSharedDir creates dir with mode 0o777 whatever the umask, as the
// emptyDir volumes a pod's containers share are: a pool pod's containers
// may run as different users and both write into it.
func MakeSharedDir(dir string) error {
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if fi.Mode().Perm() == 0o777 {
		return nil
	}
	return os.Chmod(dir, 0o777)
}

// writeAtomic replaces dir/name with data, readable by everyone, so a
// reader sees the old content or the new, never part of it.
func writeAtomic(dir, name string, data []byte) error {
	return osspecific.WriteFileAtomic(dir, name, bytes.NewReader(data), 0o644)
}

// createExclusive creates dir/name with data unless it already exists, and
// reports whether it did. The file appears with its content complete: it is
// written under a temporary name and hard-linked into place, and link(2)
// fails if the name exists.
func createExclusive(dir, name string, data []byte) (bool, error) {
	tmp, err := osspecific.CreateTemp(dir, name, bytes.NewReader(data), 0o644)
	if err != nil {
		return false, err
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := os.Link(tmp, filepath.Join(dir, name)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
