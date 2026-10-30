package poolpod

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"
)

// sweepInterval is how often Sweep looks for processes left alive.
const sweepInterval = 50 * time.Millisecond

// BecomeSubreaper makes this process the parent of every orphaned
// descendant, so a process a task daemonises (double fork, setsid) stays a
// descendant and Sweep finds it. main calls it once, before its first task.
// As the container's PID 1 it is already the reaper of last resort; this
// covers it not being PID 1. Reaping what is reparented is the caller's
// job.
func BecomeSubreaper() error {
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("become child subreaper: %w", err)
	}
	return nil
}

// Reap reaps every child of this process that has exited, without
// waiting for any that hasn't, and returns how many it reaped. main calls
// it between tasks: a task's orphans are reparented to main, and once
// Sweep has killed them they stay zombies until reaped. It must not run
// while a task's command is being waited for, which would lose its exit
// status.
func Reap() (int, error) {
	reaped := 0
	for {
		var status unix.WaitStatus
		pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
		switch {
		case errors.Is(err, unix.EINTR):
			continue
		case errors.Is(err, unix.ECHILD):
			return reaped, nil
		case err != nil:
			return reaped, fmt.Errorf("reap children: %w", err)
		case pid <= 0:
			return reaped, nil
		}
		reaped++
	}
}

// Sweep kills every descendant of this process, which in a pool pod's main
// is every process a task started, however it detached: SIGTERM to each, then
// SIGKILL to any still alive once grace has passed. It returns once none is
// left alive, or with ctx's error. A zombie counts as gone: Sweep doesn't
// reap, so it never takes an exit status the caller is waiting for.
//
// main runs it when a task is stopped, with grace half the release reserve,
// and when a task finishes, so nothing a task started outlives it.
func Sweep(ctx context.Context, grace time.Duration) error {
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return fmt.Errorf("open /proc: %w", err)
	}
	return sweep(ctx, fs, os.Getpid(), grace)
}

// sweep is Sweep over the descendants of root, as listed in fs.
func sweep(ctx context.Context, fs procfs.FS, root int, grace time.Duration) error {
	killAt := time.Now().Add(grace)
	termed := map[process]bool{}
	for {
		alive, err := descendants(fs, root)
		if err != nil {
			return err
		}
		if len(alive) == 0 {
			return nil
		}
		kill := !time.Now().Before(killAt)
		var errs []error
		for _, p := range alive {
			switch {
			case kill:
				errs = append(errs, p.signal(fs, unix.SIGKILL))
			case !termed[p]:
				termed[p] = true
				errs = append(errs, p.signal(fs, unix.SIGTERM))
			}
		}
		// A process that can't be signalled (another user's, say) would
		// be waited for in vain.
		if err := errors.Join(errs...); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sweepInterval):
		}
	}
}

// process identifies a process: its pid, and its start time so a reused pid
// isn't taken for it.
type process struct {
	pid   int
	start uint64
}

// stat is what descendants needs from /proc/<pid>/stat.
type stat struct {
	process
	ppid   int
	zombie bool
}

// descendants lists root's descendants that haven't exited.
func descendants(fs procfs.FS, root int) ([]process, error) {
	procs, err := fs.AllProcs()
	if err != nil {
		return nil, err
	}
	children := map[int][]stat{}
	for _, p := range procs {
		s, err := statOf(p)
		if isGone(err) {
			continue // exited since the directory was read
		}
		if err != nil {
			return nil, err
		}
		children[s.ppid] = append(children[s.ppid], s)
	}
	var alive []process
	queue := []int{root}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, c := range children[pid] {
			if c.pid == root {
				continue
			}
			if !c.zombie {
				alive = append(alive, c.process)
			}
			queue = append(queue, c.pid)
		}
	}
	return alive, nil
}

// readStat reads pid's /proc/<pid>/stat.
func readStat(fs procfs.FS, pid int) (stat, error) {
	p, err := fs.Proc(pid)
	if err != nil {
		return stat{}, err
	}
	return statOf(p)
}

// statOf reads p's /proc/<pid>/stat.
func statOf(p procfs.Proc) (stat, error) {
	s, err := p.Stat()
	if err != nil {
		return stat{}, fmt.Errorf("read stat for pid %d: %w", p.PID, err)
	}
	return stat{
		process: process{pid: p.PID, start: s.Starttime},
		ppid:    s.PPID,
		zombie:  s.State == "Z" || s.State == "X",
	}, nil
}

// signal sends sig to p unless p has exited. A pidfd pins the process, so
// once its start time is confirmed the signal can't reach a process that
// reused the pid; without pidfds (Linux before 5.3) it falls back to kill.
func (p process) signal(fs procfs.FS, sig unix.Signal) error {
	fd, err := unix.PidfdOpen(p.pid, 0)
	if errors.Is(err, unix.ENOSYS) {
		if killErr := ignoreGone(unix.Kill(p.pid, sig)); killErr != nil {
			return fmt.Errorf("send %v to pid %d: %w", sig, p.pid, killErr)
		}
		return nil
	}
	if err != nil {
		return ignoreGone(err)
	}
	defer func() { _ = unix.Close(fd) }()
	s, err := readStat(fs, p.pid)
	if err != nil {
		return ignoreGone(err)
	}
	if s.start != p.start {
		return nil // p exited and its pid was reused
	}
	if err := ignoreGone(unix.PidfdSendSignal(fd, sig, nil, 0)); err != nil {
		return fmt.Errorf("send %v to pid %d: %w", sig, p.pid, err)
	}
	return nil
}

// ignoreGone drops the errors that mean the process has already exited.
func ignoreGone(err error) error {
	if isGone(err) {
		return nil
	}
	return err
}

// isGone reports whether err means the process has already exited.
func isGone(err error) bool {
	return errors.Is(err, unix.ESRCH) || errors.Is(err, os.ErrNotExist)
}
