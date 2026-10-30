package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod/poolpodtest"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// The loop becomes a child subreaper and reaps every child, so it runs in a
// process of its own, TestLoopHelper, which the tests start and signal as
// the kubelet would main. The tests play the worker.

const (
	// loopHelperRootEnv is the root of the helper's shared directories.
	loopHelperRootEnv = "GO_EMISSARY_LOOP_ROOT"
	// loopHelperReserveEnv is the helper's release reserve.
	loopHelperReserveEnv = "GO_EMISSARY_LOOP_RESERVE"
	// loopWait bounds every wait for main.
	loopWait = 30 * time.Second
)

// TestLoopHelper isn't a test: it is main, started by the loop tests in a
// process of its own, as main.go runs it.
func TestLoopHelper(t *testing.T) {
	root := os.Getenv(loopHelperRootEnv)
	if root == "" {
		t.Skip("main for the loop tests, run by them in a process of its own")
	}
	os.Exit(loopHelper(root, os.Getenv(loopHelperReserveEnv)))
}

func loopHelper(root, reserve string) int {
	releaseReserve, err := time.ParseDuration(reserve)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "bad release reserve:", err)
		return 2
	}
	ctx, stop := signal.NotifyContext(logging.TestContext(context.Background()), syscall.SIGTERM)
	defer stop()
	tracer, _, err := tracing.CreateDefaultTestTracing(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "tracing:", err)
		return 2
	}
	err = runLoop(ctx, loopConfig{
		Base:           loopBase(root),
		ReleaseReserve: releaseReserve,
		Environ:        os.Environ(),
		Driver:         maindriver.Container{},
		Tracer:         tracer,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "loop failed:", err)
		return 1
	}
	return 0
}

func loopBase(root string) wfexecutor.TaskPaths {
	return wfexecutor.TaskPaths{
		VarRunArgo:     filepath.Join(root, "var/run/argo"),
		InputArtifacts: filepath.Join(root, "argo/inputs/artifacts"),
		Staging:        filepath.Join(root, "argo/staging"),
		Scratch:        filepath.Join(root, "tmp/argo"),
	}
}

// loopHarness is a pool pod for the loop tests: main is TestLoopHelper,
// and the test is the worker.
type loopHarness struct {
	t       *testing.T
	root    string
	base    wfexecutor.TaskPaths
	pool    poolpod.Pool
	reserve time.Duration
	// image stands for main's own filesystem, from its image: the paths
	// tasks read and write outside volumes.
	image string
	// out is where tasks leave what the test checks, beyond the loop's
	// reach, as on a volume.
	out string
	log string

	main   *exec.Cmd
	exited chan error
}

func newLoopHarness(t *testing.T) *loopHarness {
	t.Helper()
	root := t.TempDir()
	h := &loopHarness{
		t:       t,
		root:    root,
		base:    loopBase(root),
		reserve: 2 * time.Second,
		image:   filepath.Join(root, "image"),
		out:     filepath.Join(root, "out"),
		log:     filepath.Join(root, "main.log"),
	}
	h.pool = poolpod.NewPool(filepath.Join(h.base.VarRunArgo, poolpod.DirName))
	for _, dir := range []string{h.base.VarRunArgo, h.base.InputArtifacts, h.base.Staging, h.image, h.out} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	t.Cleanup(func() {
		if t.Failed() {
			if log, err := os.ReadFile(h.log); err == nil {
				t.Logf("main's log:\n%s", log)
			}
		}
	})
	return h
}

// start starts main, as the kubelet starts or restarts its container.
func (h *loopHarness) start() {
	h.t.Helper()
	log, err := os.OpenFile(h.log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(h.t, err)
	defer func() { _ = log.Close() }()
	main := exec.CommandContext(h.t.Context(), os.Args[0], "-test.run=^TestLoopHelper$")
	main.Env = append(os.Environ(), loopHelperRootEnv+"="+h.root, loopHelperReserveEnv+"="+h.reserve.String())
	main.Stdout, main.Stderr = log, log
	require.NoError(h.t, main.Start())
	exited := make(chan error, 1)
	go func() { exited <- main.Wait() }()
	h.main, h.exited = main, exited
	h.t.Cleanup(func() {
		_ = main.Process.Kill()
		h.t.Logf("main exited: %v", <-exited)
	})
}

// signal sends sig to main.
func (h *loopHarness) signal(sig syscall.Signal) {
	h.t.Helper()
	require.NoError(h.t, h.main.Process.Signal(sig))
}

// waitExit waits up to timeout for main to exit, and returns how it exited.
func (h *loopHarness) waitExit(timeout time.Duration) error {
	h.t.Helper()
	select {
	case err := <-h.exited:
		h.exited <- err
		return err
	case <-time.After(timeout):
		require.FailNow(h.t, "main didn't exit", "within %s", timeout)
		return nil
	}
}

// waitLog waits for main to log a line containing all of parts.
func (h *loopHarness) waitLog(parts ...string) {
	h.t.Helper()
	require.Eventually(h.t, func() bool {
		log, err := os.ReadFile(h.log)
		if err != nil {
			return false
		}
		for line := range strings.SplitSeq(string(log), "\n") {
			if containsAll(line, parts) {
				return true
			}
		}
		return false
	}, loopWait, 20*time.Millisecond, "main never logged %q", parts)
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func (h *loopHarness) paths(name string) wfexecutor.TaskPaths {
	h.t.Helper()
	paths, err := h.base.PoolTask(name)
	require.NoError(h.t, err)
	return paths
}

// spec is a task that runs script with sh, with the harness's directories
// in its env as IMAGE and OUT. main expands the script as the kubelet
// expands args, so a $$ for sh is written $$$$.
func (h *loopHarness) spec(name string, tmpl *wfv1.Template, script string) poolpod.TaskSpec {
	if tmpl == nil {
		tmpl = &wfv1.Template{}
	}
	return poolpod.TaskSpec{
		NodeID:   name,
		Template: tmpl,
		Command:  []string{"sh", "-c"},
		Args:     []string{script},
		Env:      []poolpod.EnvVar{{Name: "IMAGE", Value: h.image}, {Name: "OUT", Value: h.out}},
	}
}

// offer stages inputs for the task called name and offers it.
func (h *loopHarness) offer(name string, spec poolpod.TaskSpec, inputs map[string]string) poolpod.Task {
	h.t.Helper()
	paths := h.paths(name)
	for artifact, content := range inputs {
		poolpodtest.StageInput(h.t, paths.InputArtifacts, artifact, content)
	}
	return poolpodtest.Offer(h.t, h.pool, paths.VarRunArgo, spec)
}

func (h *loopHarness) wait(task poolpod.Task) poolpod.Status {
	h.t.Helper()
	return poolpodtest.WaitMain(h.t, task, loopWait)
}

// waitFile waits for a task to create path.
func (h *loopHarness) waitFile(path string) {
	h.t.Helper()
	require.Eventually(h.t, func() bool {
		_, err := os.Stat(path)
		return err == nil
	}, loopWait, 10*time.Millisecond, "%s never appeared", path)
}

func readPid(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	return pid
}

// gone reports whether pid has exited and been reaped.
func gone(pid int) bool {
	_, err := os.Stat("/proc/" + strconv.Itoa(pid))
	return os.IsNotExist(err)
}

func exited(code int) poolpod.Status {
	return poolpod.Status{State: poolpod.Exited, ExitCode: code}
}

// Two tasks back to back with the same paths: the second sees none of the
// first's inputs, outputs or processes, and each stages its own outputs.
func TestLoop_TasksBackToBackShareNothing(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	in := filepath.Join(h.image, "in", "data")
	out := filepath.Join(h.image, "out", "result")
	tmpl := &wfv1.Template{
		Inputs:  wfv1.Inputs{Artifacts: wfv1.Artifacts{{Name: "in", Path: in}}},
		Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "out", ValueFrom: &wfv1.ValueFrom{Path: out}}}},
	}
	// Each task checks it starts clean: no output from before, its own
	// input, and no process the first started.
	script := func(name string) string {
		return `set -e
test ! -e "$IMAGE/out"
test "$(readlink "$IMAGE/in/data")" = "` + filepath.Join(h.paths(name).InputArtifacts, "in") + `"
if [ -s "$OUT/bg-one" ] && kill -0 "$(cat "$OUT/bg-one")" 2>/dev/null; then exit 11; fi
mkdir "$IMAGE/out"
cat "$IMAGE/in/data" > "$IMAGE/out/result"
sleep 300 > /dev/null 2>&1 &
echo $! > "$OUT/bg-` + name + `"
echo "ran ` + name + `"`
	}
	h.start()

	one := h.spec("one", tmpl, script("one"))
	one.IncludeScriptOutput = true
	first := h.offer("one", one, map[string]string{"in": "first"})
	require.Equal(t, exited(0), h.wait(first))
	staged, err := os.ReadFile(filepath.Join(h.paths("one").VarRunArgo, "outputs", "parameters", out))
	require.NoError(t, err)
	assert.Equal(t, "first", string(staged))
	stdout, err := os.ReadFile(filepath.Join(h.paths("one").ContainerDir(common.MainContainerName), "stdout"))
	require.NoError(t, err)
	assert.Equal(t, "ran one\n", string(stdout), "logs are captured in the task's own directory")
	assert.NoDirExists(t, filepath.Join(h.image, "in"), "the input link and the directory made for it are removed")
	assert.NoDirExists(t, filepath.Join(h.image, "out"), "the output and the directory the task made for it are removed")
	background := readPid(t, filepath.Join(h.out, "bg-one"))
	assert.Eventually(t, func() bool { return gone(background) }, loopWait, 20*time.Millisecond, "a background process outlived its task")

	second := h.offer("two", h.spec("two", tmpl, script("two")), map[string]string{"in": "second"})
	require.Equal(t, exited(0), h.wait(second))
	staged, err = os.ReadFile(filepath.Join(h.paths("two").VarRunArgo, "outputs", "parameters", out))
	require.NoError(t, err)
	assert.Equal(t, "second", string(staged))
}

// An input or declared output path that exists in the image fails the task
// before it starts, and the image's file is left as it was.
func TestLoop_PathInTheImageFailsTheTask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	existing := filepath.Join(h.image, "existing")
	require.NoError(t, os.WriteFile(existing, []byte("from the image"), 0o644))
	h.start()

	for name, tmpl := range map[string]*wfv1.Template{
		"input-artifact":   {Inputs: wfv1.Inputs{Artifacts: wfv1.Artifacts{{Name: "in", Path: existing}}}},
		"output-artifact":  {Outputs: wfv1.Outputs{Artifacts: wfv1.Artifacts{{Name: "out", Path: existing}}}},
		"output-parameter": {Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "out", ValueFrom: &wfv1.ValueFrom{Path: existing}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			task := h.offer(name, h.spec(name, tmpl, `touch "$OUT/ran-`+name+`"`), map[string]string{"in": "staged"})
			s := h.wait(task)
			assert.Equal(t, poolpod.Exited, s.State)
			assert.Equal(t, loopExitCodeMainFailed, s.ExitCode)
			assert.Contains(t, s.Message, existing+" already exists in the container image")
			assert.NoFileExists(t, filepath.Join(h.out, "ran-"+name), "the task must not start")
			content, err := os.ReadFile(existing)
			require.NoError(t, err)
			assert.Equal(t, "from the image", string(content))
		})
	}
}

// The kubelet's SIGTERM doesn't stop a running task: main finishes it, then
// exits.
func TestLoop_SIGTERMDoesNotStopATask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	task := h.offer("t", h.spec("t", nil, `touch "$OUT/started"; sleep 2; touch "$OUT/finished"`), nil)
	h.waitFile(filepath.Join(h.out, "started"))

	h.signal(syscall.SIGTERM)

	require.Equal(t, exited(0), h.wait(task))
	assert.FileExists(t, filepath.Join(h.out, "finished"))
	require.NoError(t, h.waitExit(loopWait), "with the task done, main exits")
}

// A task main takes as SIGTERM lands is run, not abandoned: the worker
// would see main as having died in it.
func TestLoop_TakeAfterSIGTERMRunsTheTask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	require.NoError(t, h.pool.Init())
	task := h.offer("t", h.spec("t", nil, `true`), nil)
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	cancel()

	run, err := take(ctx, task)
	require.NoError(t, err)
	require.NotNil(t, run, "main must run the task it took")
	require.NoError(t, run.Finish(0, ""))
	assert.Equal(t, exited(0), h.wait(task))
}

// Each task's command runs under a span of its own, a child of the task's
// trace context.
func TestLoop_TracesTheTask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	const (
		traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		parent  = "00f067aa0ba902b7"
	)
	spec := h.spec("traced", nil, `echo "$TRACEPARENT" > "$OUT/traceparent"`)
	spec.Env = append(spec.Env, poolpod.EnvVar{Name: "TRACEPARENT", Value: "00-" + traceID + "-" + parent + "-01"})
	task := h.offer("traced", spec, nil)
	require.Equal(t, exited(0), h.wait(task))
	got, err := os.ReadFile(filepath.Join(h.out, "traceparent"))
	require.NoError(t, err)
	fields := strings.Split(strings.TrimSpace(string(got)), "-")
	require.Len(t, fields, 4, "TRACEPARENT %q", got)
	assert.Equal(t, traceID, fields[1], "the task's trace")
	assert.NotEqual(t, parent, fields[2], "the command's parent is main's span, not the task's")
}

// stop stops a running task: SIGTERM, then SIGKILL after half the release
// reserve for what ignores it. main then takes the next task.
func TestLoop_StopKillsTheTask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()

	t.Run("SIGTERM", func(t *testing.T) {
		task := h.offer("term", h.spec("term", nil, `touch "$OUT/term"; exec sleep 300`), nil)
		h.waitFile(filepath.Join(h.out, "term"))
		_, err := task.Stop(poolpod.StopInterrupt)
		require.NoError(t, err)
		assert.Equal(t, exited(128+int(syscall.SIGTERM)), h.wait(task))
	})
	t.Run("SIGKILL", func(t *testing.T) {
		task := h.offer("kill", h.spec("kill", nil, `trap "" TERM; touch "$OUT/kill"; sleep 300; true`), nil)
		h.waitFile(filepath.Join(h.out, "kill"))
		stopped := time.Now()
		_, err := task.Stop(poolpod.StopDeadline)
		require.NoError(t, err)
		assert.Equal(t, exited(128+int(syscall.SIGKILL)), h.wait(task))
		assert.GreaterOrEqual(t, time.Since(stopped), h.reserve/2, "SIGKILL only after half the release reserve")
	})
}

// A task the worker stopped before main took it never starts.
func TestLoop_StoppedBeforeItStarts(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	task := poolpodtest.Prepare(t, h.paths("t").VarRunArgo, h.spec("t", nil, `touch "$OUT/ran"`))
	_, err := task.Stop(poolpod.StopRelease)
	require.NoError(t, err)
	require.NoError(t, h.pool.Offer(task))

	assert.Equal(t, exited(loopExitCodeStopped), h.wait(task))
	assert.NoFileExists(t, filepath.Join(h.out, "ran"))
}

// A restarted main finds the task its predecessor died in taken, and never
// runs it again.
func TestLoop_RestartDoesNotRerunATakenTask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	task := h.offer("t", h.spec("t", nil, `echo run >> "$OUT/runs"; echo $$$$ > "$OUT/pid"; exec sleep 300`), nil)
	poolpodtest.WaitRunning(t, task, loopWait)
	h.waitFile(filepath.Join(h.out, "pid"))
	// In a pod, the container's processes die with it.
	orphan := readPid(t, filepath.Join(h.out, "pid"))
	t.Cleanup(func() { _ = syscall.Kill(orphan, syscall.SIGKILL) })

	h.signal(syscall.SIGKILL)
	require.Error(t, h.waitExit(loopWait))
	require.NoError(t, syscall.Kill(orphan, syscall.SIGKILL))
	require.Equal(t, poolpod.Status{State: poolpod.Died}, h.wait(task))

	h.start()
	h.waitLog("skipping task", task.Dir())
	next := h.offer("next", h.spec("next", nil, `true`), nil)
	require.Equal(t, exited(0), h.wait(next))

	runs, err := os.ReadFile(filepath.Join(h.out, "runs"))
	require.NoError(t, err)
	assert.Equal(t, "run\n", string(runs))
	s, err := task.Status()
	require.NoError(t, err)
	assert.Equal(t, poolpod.Status{State: poolpod.Died}, s)
}

// A process a task daemonises in a session of its own is gone before the
// next task starts.
func TestLoop_DaemonDoesNotOutliveItsTask(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("needs setsid")
	}
	h := newLoopHarness(t)
	h.start()
	daemonPid := filepath.Join(h.out, "daemon")
	task := h.offer("daemon", h.spec("daemon", nil, `setsid sh -c 'echo $$$$ > "$OUT/daemon"; exec sleep 300' > /dev/null 2>&1 &
while [ ! -s "$OUT/daemon" ]; do sleep 0.05; done`), nil)
	require.Equal(t, exited(0), h.wait(task))
	daemon := readPid(t, daemonPid)
	t.Cleanup(func() { _ = syscall.Kill(daemon, syscall.SIGKILL) })

	next := h.offer("next", h.spec("next", nil, `! kill -0 "$(cat "$OUT/daemon")" 2>/dev/null`), nil)
	assert.Equal(t, exited(0), h.wait(next), "the daemon was still running when the next task started")
	assert.True(t, gone(daemon))
}

// With no task taken, SIGTERM makes main exit at once.
func TestLoop_SIGTERMWithNoTaskExitsAtOnce(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	// The loop has created the pool directory once SIGTERM is handled.
	h.waitFile(h.pool.Dir())

	h.signal(syscall.SIGTERM)

	require.NoError(t, h.waitExit(5*time.Second))
}

// main holds its liveness lock from when it starts taking tasks until it
// exits, so the worker claims tasks only while main can run them.
func TestLoop_HoldsMainsLockWhileUp(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	up, err := h.pool.MainUp()
	require.NoError(t, err)
	require.False(t, up)
	h.start()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	require.NoError(t, h.pool.WaitMainUp(ctx))

	h.signal(syscall.SIGTERM)

	require.NoError(t, h.waitExit(5*time.Second))
	up, err = h.pool.MainUp()
	require.NoError(t, err)
	assert.False(t, up)
}

// A task the worker withdrew before main took it is skipped, its lock
// released, and main takes the next.
func TestLoop_SkipsAWithdrawnTask(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	task := poolpodtest.Prepare(t, h.paths("withdrawn").VarRunArgo, h.spec("withdrawn", nil, `touch "$OUT/ran"`))
	withdrew, err := task.Withdraw()
	require.NoError(t, err)
	require.True(t, withdrew)
	require.NoError(t, h.pool.Offer(task))
	h.waitLog("skipping task", task.Dir(), poolpod.ErrWithdrawn.Error())

	held, err := osspecific.LockHeld(filepath.Join(task.Dir(), poolpod.LockFile))
	require.NoError(t, err)
	assert.False(t, held)
	next := h.offer("next", h.spec("next", nil, `true`), nil)
	require.Equal(t, exited(0), h.wait(next))
	assert.NoFileExists(t, filepath.Join(h.out, "ran"))
	assert.Equal(t, poolpod.Status{State: poolpod.Withdrawn}, poolpodtest.WaitMain(t, task, loopWait))
}

// The task's env is expanded as the kubelet expands a container's, over
// main's own.
func TestLoop_ExpandsTheTaskEnv(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	spec := h.spec("env", nil, `echo "$(GREETING) $GREETING" > "$OUT/env"`)
	spec.Env = append(spec.Env, poolpod.EnvVar{Name: "GREETING", Value: "hello from $(" + loopHelperRootEnv + ")"})
	task := h.offer("env", spec, nil)
	require.Equal(t, exited(0), h.wait(task))
	got, err := os.ReadFile(filepath.Join(h.out, "env"))
	require.NoError(t, err)
	want := "hello from " + h.root
	assert.Equal(t, want+" "+want+"\n", string(got))
}

// A task main can't run fails with main's reason, and never with exit code
// 0.
func TestLoop_TaskMainCannotRun(t *testing.T) {
	t.Parallel()
	h := newLoopHarness(t)
	h.start()
	spec := h.spec("missing", nil, "")
	spec.Command, spec.Args = []string{"/no/such/command"}, nil
	task := h.offer("missing", spec, nil)
	s := h.wait(task)
	assert.Equal(t, poolpod.Exited, s.State)
	assert.Equal(t, loopExitCodeMainFailed, s.ExitCode)
	assert.Contains(t, s.Message, "/no/such/command")
}
