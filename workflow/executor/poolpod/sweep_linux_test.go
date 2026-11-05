package poolpod

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// These tests sweep the test process's own descendants, which are only the
// processes they start: none of the package's tests run in parallel.

// realProc is the real /proc.
func realProc(t *testing.T) procfs.FS {
	t.Helper()
	fs, err := procfs.NewDefaultFS()
	require.NoError(t, err)
	return fs
}

// alive reports whether pid is a process that hasn't exited.
func alive(t *testing.T, pid int) bool {
	t.Helper()
	s, err := readStat(realProc(t), pid)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	require.NoError(t, err)
	return !s.zombie
}

// A task that daemonises a child (setsid) leaves no process behind.
func TestSweep_KillsADaemonisedChild(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("needs setsid")
	}
	require.NoError(t, BecomeSubreaper())

	pidFile := filepath.Join(t.TempDir(), "pid")
	// The task's shell exits at once, leaving a daemon in its own session.
	task := exec.CommandContext(t.Context(), "sh", "-c", `setsid sh -c 'echo $$ > "$1"; exec sleep 300' daemon "$0" &`, pidFile)
	require.NoError(t, task.Run())
	var pid int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(pidFile)
		if err != nil || !strings.HasSuffix(string(data), "\n") {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)
	t.Cleanup(func() {
		_ = unix.Kill(pid, unix.SIGKILL)
		var ws unix.WaitStatus
		_, _ = unix.Wait4(pid, &ws, 0, nil)
	})
	s, err := readStat(realProc(t), pid)
	require.NoError(t, err)
	require.Equal(t, os.Getpid(), s.ppid, "the daemon is reparented to the subreaper")
	sid, err := unix.Getsid(pid)
	require.NoError(t, err)
	require.Equal(t, pid, sid, "the daemon leads its own session")

	require.NoError(t, Sweep(t.Context(), 5*time.Second))
	assert.False(t, alive(t, pid))

	var ws unix.WaitStatus
	_, err = unix.Wait4(pid, &ws, 0, nil)
	require.NoError(t, err)
	assert.True(t, ws.Signaled())
	assert.Equal(t, unix.SIGTERM, ws.Signal())
}

func TestSweep_KillsWhatIgnoresSIGTERM(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "GO_POOLPOD_HELPER=ignore-term") //nolint:forbidigo // env is the only channel to a re-exec'd test helper process
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "READY\n", line)

	const grace = 300 * time.Millisecond
	start := time.Now()
	require.NoError(t, Sweep(t.Context(), grace))
	assert.GreaterOrEqual(t, time.Since(start), grace, "SIGKILL waits for the grace period")

	err = cmd.Wait()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	ws := exitErr.Sys().(syscall.WaitStatus)
	assert.True(t, ws.Signaled())
	assert.Equal(t, syscall.SIGKILL, ws.Signal())
}

func TestSweep_DoesNotWaitOutTheGraceForWhatExits(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "300")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	start := time.Now()
	done := make(chan error, 1)
	go func() { done <- Sweep(t.Context(), time.Minute) }()
	// sleep is a zombie once SIGTERM kills it, which counts as gone.
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Sweep did not return once its only process had exited")
	}
	assert.Less(t, time.Since(start), 10*time.Second)

	err := cmd.Wait()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, syscall.SIGTERM, exitErr.Sys().(syscall.WaitStatus).Signal())
}

func TestSweep_NothingToSweep(t *testing.T) {
	require.NoError(t, Sweep(t.Context(), time.Minute))
}

// fakeProc writes a /proc-like directory with a stat file per entry.
func fakeProc(t *testing.T, stats map[int]string) procfs.FS {
	t.Helper()
	dir := t.TempDir()
	for pid, s := range stats {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, strconv.Itoa(pid)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, strconv.Itoa(pid), "stat"), []byte(s), 0o644))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "self"), 0o755))
	fs, err := procfs.NewFS(dir)
	require.NoError(t, err)
	return fs
}

// fakeStat is a /proc/<pid>/stat line with the given name, state, parent
// and start time.
func fakeStat(pid int, name, state string, ppid int, start uint64) string {
	// Fields 5-21 and 23-52 are zero; 22 is the start time.
	fields := slices.Concat(
		[]string{strconv.Itoa(pid), "(" + name + ")", state, strconv.Itoa(ppid)},
		slices.Repeat([]string{"0"}, 17),
		[]string{strconv.FormatUint(start, 10)},
		slices.Repeat([]string{"0"}, 30),
	)
	return strings.Join(fields, " ") + "\n"
}

func TestReadStat(t *testing.T) {
	dir := fakeProc(t, map[int]string{
		10: fakeStat(10, "a) b (c", "S", 7, 1234),
		11: fakeStat(11, "zombie", "Z", 10, 99),
		12: "12 (truncated) S 1",
	})
	s, err := readStat(dir, 10)
	require.NoError(t, err)
	assert.Equal(t, stat{process: process{pid: 10, start: 1234}, ppid: 7}, s, "a name holding spaces and parentheses")

	s, err = readStat(dir, 11)
	require.NoError(t, err)
	assert.True(t, s.zombie)

	_, err = readStat(dir, 12)
	require.Error(t, err)

	_, err = readStat(realProc(t), os.Getpid())
	require.NoError(t, err, "the real format")
}

func TestDescendants(t *testing.T) {
	dir := fakeProc(t, map[int]string{
		1:   fakeStat(1, "init", "S", 0, 1),
		100: fakeStat(100, "main", "S", 1, 2),
		101: fakeStat(101, "task", "S", 100, 3),
		102: fakeStat(102, "grandchild", "R", 101, 4),
		103: fakeStat(103, "exited", "Z", 100, 5),
		104: fakeStat(104, "exec'd in", "S", 0, 6),
		105: fakeStat(105, "init's", "S", 1, 7),
	})
	got, err := descendants(dir, 100)
	require.NoError(t, err)
	assert.ElementsMatch(t, []process{{pid: 101, start: 3}, {pid: 102, start: 4}}, got)
}

func TestSweep_Cancellation(t *testing.T) {
	// A descendant that never goes away: a pid no process has, so
	// signalling it finds nothing.
	pid := 1 << 22
	for alive(t, pid) {
		pid--
	}
	dir := fakeProc(t, map[int]string{pid: fakeStat(pid, "stuck", "D", 100, 1)})

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	err := sweep(ctx, dir, 100, 0)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "1 processes still alive")
}

// A process whose pid now has another start time has exited and its pid
// been reused: the process now holding the pid isn't signalled.
func TestProcessSignal_SkipsAReusedPid(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sleep", "300")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	pid := cmd.Process.Pid
	dir := fakeProc(t, map[int]string{pid: fakeStat(pid, "sleep", "S", os.Getpid(), 2)})

	require.NoError(t, process{pid: pid, start: 1}.signal(dir, unix.SIGKILL))
	assert.Never(t, func() bool { return !alive(t, pid) }, 200*time.Millisecond, 10*time.Millisecond, "the pid's new process was signalled")

	require.NoError(t, process{pid: pid, start: 2}.signal(dir, unix.SIGTERM))
	err := cmd.Wait()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, syscall.SIGTERM, exitErr.Sys().(syscall.WaitStatus).Signal())
}
