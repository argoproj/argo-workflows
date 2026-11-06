package commands

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	utilretry "github.com/argoproj/argo-workflows/v4/util/retry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod/poolpodtest"
)

// The worker tests run a pool pod: main is the loop tests' TestLoopHelper,
// in a process of its own, and the worker is `argoexec worker` in the test
// process, claiming its tasks from a dispatch fake as the controller
// enqueues them.

const workerNamespace = "argo"

// workerHarness is a pool pod whose worker the test runs, and the
// controller's side of its pool.
type workerHarness struct {
	*loopHarness
	fake   *dispatch.Fake
	worker dispatch.Worker
	// grace is the pod's grace period. What the worker keeps back at its
	// end to release a task is main's reserve: as in a pool pod, the two
	// are the same.
	grace time.Duration
	// logs is what every run of the worker logged.
	logs *logging.TestHook
}

func newWorkerHarness(t *testing.T) *workerHarness {
	t.Helper()
	lh := newLoopHarness(t)
	return &workerHarness{
		loopHarness: lh,
		fake:        dispatch.NewFake(),
		worker:      dispatch.Worker{Namespace: workerNamespace, PoolHash: "pool-hash", PodName: "pool-pod", PodUID: "pool-pod-uid"},
		grace:       30 * time.Second,
		logs:        logging.NewTestHook(),
	}
}

// workerRun is one run of the worker container.
type workerRun struct {
	t      *testing.T
	cancel context.CancelFunc
	done   chan error
}

// startWorker starts a run of the worker, whose result writes give up
// after retryTimeout.
func (h *workerHarness) startWorker(retryTimeout time.Duration) *workerRun {
	h.t.Helper()
	logger := logging.NewSlogLogger(logging.Info, logging.Text, h.logs).WithField("test", "true")
	ctx, cancel := context.WithCancel(logging.WithLogger(h.t.Context(), logger))
	w := &argoexecexecutor.Worker{Consumer: h.fake.Consumer(h.worker), Base: h.base, PodName: h.worker.PodName}
	process, err := wfexecutor.NewProcess(ctx, nil, nil, nil,
		func(paths wfexecutor.TaskPaths) wfexecutor.ContainerRuntimeExecutor {
			return emissary.New(paths.VarRunArgo)
		},
		wfexecutor.ProcessConfig{
			PodName:                h.worker.PodName,
			PodUID:                 h.worker.PodUID,
			Namespace:              workerNamespace,
			RetryBackoff:           utilretry.ExecutorRetry(ctx),
			TaskResultRetryTimeout: retryTimeout,
			TerminationGracePeriod: h.grace,
			ReleaseReserve:         h.reserve,
			StopWorker:             w.Stop,
		})
	require.NoError(h.t, err)
	w.Process = process
	r := &workerRun{t: h.t, cancel: cancel, done: make(chan error, 1)}
	go func() { r.done <- w.Run(ctx) }()
	h.t.Cleanup(cancel)
	return r
}

// terminate stops the worker as SIGTERM does, and returns how it exited.
func (r *workerRun) terminate() error {
	r.t.Helper()
	r.cancel()
	return r.wait()
}

// wait waits for the worker to exit, and returns how it did.
func (r *workerRun) wait() error {
	r.t.Helper()
	select {
	case err := <-r.done:
		return err
	case <-time.After(loopWait):
		require.FailNow(r.t, "the worker didn't exit")
		return nil
	}
}

// waitWorkerLog waits for a run of the worker to log msg.
func (h *workerHarness) waitWorkerLog(msg string) {
	h.t.Helper()
	require.Eventually(h.t, func() bool {
		return slices.ContainsFunc(h.logs.AllEntries(), func(e logging.TestEntry) bool { return e.Msg == msg })
	}, loopWait, 20*time.Millisecond, "the worker never logged %q", msg)
}

// task is a pooled task running script with sh, with the harness's
// directories in its env as IMAGE and OUT.
func (h *workerHarness) task(nodeID string, tmpl wfv1.Template, script string) dispatch.Task {
	tmpl.Name = "t"
	tmpl.Script = &wfv1.ScriptTemplate{
		Container: apiv1.Container{
			Image:        "busybox",
			Command:      []string{"sh"},
			Env:          []apiv1.EnvVar{{Name: "IMAGE", Value: h.image}, {Name: "OUT", Value: h.out}},
			VolumeMounts: tmpl.GetVolumeMounts(),
		},
		Source: "set -e\n" + script,
	}
	tmpl.Container = nil
	return dispatch.Task{
		Key:          dispatch.Key{Namespace: workerNamespace, NodeID: nodeID},
		WorkflowName: "wf",
		WorkflowUID:  "wf-uid",
		PoolHash:     h.worker.PoolHash,
		Template:     tmpl,
		Pool:         wfv1.PoolSpec{Name: "pool", InitialReplicas: 1},
		Dispatch:     wfv1.TaskDispatch{TaskName: "wf-" + nodeID},
	}
}

func (h *workerHarness) enqueue(task dispatch.Task) {
	h.t.Helper()
	_, err := h.fake.Dispatcher().Enqueue(logging.TestContext(h.t.Context()), task)
	require.NoError(h.t, err)
}

// completed waits for the task's claim to be completed, and returns it.
func (h *workerHarness) completed(nodeID string) *wfv1.WorkflowTaskResult {
	h.t.Helper()
	var result *wfv1.WorkflowTaskResult
	require.Eventually(h.t, func() bool {
		r, ok := h.fake.Result(workerNamespace, nodeID)
		result = r
		return ok && r.Labels[common.LabelKeyReportOutputsCompleted] == "true"
	}, loopWait, 20*time.Millisecond, "%s was never completed", nodeID)
	return result
}

// claim claims the only candidate as the worker, as a run of the worker
// that then stopped did.
func (h *workerHarness) claim() dispatch.Claimed {
	h.t.Helper()
	ctx := logging.TestContext(h.t.Context())
	consumer := h.fake.Consumer(h.worker)
	candidates, err := consumer.Candidates(ctx)
	require.NoError(h.t, err)
	require.Len(h.t, candidates, 1)
	claimed, err := consumer.Claim(ctx, candidates[0])
	require.NoError(h.t, err)
	return claimed
}

// poolEntries lists what is in each of the pool pod's shared directories
// that tasks have directories of their own beneath.
func (h *workerHarness) poolEntries() map[string][]string {
	h.t.Helper()
	got := map[string][]string{}
	for _, dir := range []string{filepath.Join(h.base.VarRunArgo, poolpod.DirName), h.base.InputArtifacts, h.base.Staging, filepath.Join(h.base.Scratch, "outputs")} {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(h.t, err)
		for _, e := range entries {
			got[dir] = append(got[dir], e.Name())
		}
	}
	return got
}

// assertPoolClean checks no task's directories are left, only the pool
// directory's current, if a task was offered, and main's liveness lock.
func (h *workerHarness) assertPoolClean() {
	h.t.Helper()
	entries := h.poolEntries()
	pool := filepath.Join(h.base.VarRunArgo, poolpod.DirName)
	entries[pool] = slices.DeleteFunc(entries[pool], func(e string) bool {
		return e == poolpod.CurrentFile || e == poolpod.MainFile
	})
	if len(entries[pool]) == 0 {
		delete(entries, pool)
	}
	assert.Empty(h.t, entries)
}

// artifactServer serves input artifacts and stores output artifacts, over
// HTTP.
type artifactServer struct {
	*httptest.Server
	mu  sync.Mutex
	put map[string]string
}

func newArtifactServer(t *testing.T, get map[string]string) *artifactServer {
	s := &artifactServer{put: map[string]string{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			body, ok := get[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, body)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.mu.Lock()
			s.put[r.URL.Path] = string(body)
			s.mu.Unlock()
		default:
			http.Error(w, r.Method, http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *artifactServer) stored(path string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.put[path]
	return body, ok
}

// A task runs end to end: its input artifacts are staged, main runs it with
// its parameter, and its result, output parameter and output artifacts,
// from the image and from a volume, are reported through its claim. Then
// nothing of it is left.
func TestWorker_RunsATaskEndToEnd(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	srv := newArtifactServer(t, map[string]string{"/in": "from http"})
	work := filepath.Join(h.root, "work")
	require.NoError(t, os.MkdirAll(work, 0o755))
	tmpl := wfv1.Template{
		Volumes: []apiv1.Volume{{Name: "work", VolumeSource: apiv1.VolumeSource{HostPath: &apiv1.HostPathVolumeSource{Path: work}}}},
		Script:  &wfv1.ScriptTemplate{Container: apiv1.Container{VolumeMounts: []apiv1.VolumeMount{{Name: "work", MountPath: work}}}},
		Inputs: wfv1.Inputs{
			Parameters: []wfv1.Parameter{{Name: "greeting", Value: wfv1.AnyStringPtr("hello")}},
			Artifacts: wfv1.Artifacts{
				{Name: "raw", Path: filepath.Join(h.image, "in", "raw"), ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "from raw"}}},
				{Name: "http", Path: filepath.Join(h.image, "in", "http"), ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/in"}}},
			},
		},
		Outputs: wfv1.Outputs{
			Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: filepath.Join(h.image, "out", "p")}}},
			Artifacts: wfv1.Artifacts{
				{Name: "image", Path: filepath.Join(h.image, "out", "a"), ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/image"}}},
				{Name: "volume", Path: filepath.Join(work, "v"), Archive: &wfv1.ArchiveStrategy{None: &wfv1.NoneStrategy{}}, ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/volume"}}},
			},
		},
	}
	// The input parameter, greeting, as the controller substituted it.
	task := h.task("node-1", tmpl, `
mkdir -p "$IMAGE/out"
printf 'hello %s %s' "$(cat "$IMAGE/in/raw")" "$(cat "$IMAGE/in/http")" > "$IMAGE/out/p"
echo artifact > "$IMAGE/out/a"
echo volume > "`+work+`/v"
echo -n result`)
	task.Dispatch.IncludeScriptOutput = true
	h.start()
	run := h.startWorker(time.Minute)
	h.enqueue(task)

	r := h.completed("node-1")
	assert.Equal(t, wfv1.NodeSucceeded, r.Phase, r.Message)
	require.NotNil(t, r.Outputs)
	require.NotNil(t, r.Outputs.Result)
	assert.Equal(t, "result", *r.Outputs.Result)
	require.NotNil(t, r.Outputs.ExitCode)
	assert.Equal(t, "0", *r.Outputs.ExitCode)
	require.Len(t, r.Outputs.Parameters, 1)
	assert.Equal(t, "hello from raw from http", r.Outputs.Parameters[0].Value.String())
	var saved []string
	for _, a := range r.Outputs.Artifacts {
		saved = append(saved, a.Name)
	}
	assert.ElementsMatch(t, []string{"image", "volume"}, saved)
	image, ok := srv.stored("/image")
	require.True(t, ok, "the artifact main staged from the image is uploaded")
	assert.NotEmpty(t, image)
	volume, ok := srv.stored("/volume")
	require.True(t, ok)
	assert.Equal(t, "volume\n", volume, "an artifact on a volume is uploaded as main left it")

	require.NoError(t, run.terminate(), "with no task in hand, SIGTERM is a clean exit")
	h.assertPoolClean()
	assert.NoDirExists(t, filepath.Join(h.image, "in"))
	assert.NoDirExists(t, filepath.Join(h.image, "out"))
}

// A worker whose main hasn't started, as when main's image can't be
// pulled, claims nothing, so the pool's tasks stay queued, and takes them
// once main is up.
func TestWorker_WaitsForMain(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	run := h.startWorker(time.Minute)
	h.enqueue(h.task("node-1", wfv1.Template{}, `true`))

	h.waitWorkerLog("Waiting for main to start before taking a task")
	_, claimed := h.fake.Result(workerNamespace, "node-1")
	assert.False(t, claimed, "claimed with no main to run it")

	h.start()
	assert.Equal(t, wfv1.NodeSucceeded, h.completed("node-1").Phase)
	require.NoError(t, run.terminate())
}

func dirExists(dir string) bool {
	_, err := os.Stat(dir)
	return err == nil
}

// A worker removes what an earlier run left in the pool before it claims
// anything, and exits, claiming nothing, if it can't, so its pod crash
// loops rather than run tasks that could see it.
func TestWorker_ClearsThePoolAtStart(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("needs a user that directory permissions apply to")
	}
	h := newWorkerHarness(t)
	leftover := filepath.Join(h.base.Staging, "old-task", "script")
	require.NoError(t, os.MkdirAll(filepath.Dir(leftover), 0o755))
	require.NoError(t, os.WriteFile(leftover, nil, 0o644))
	require.NoError(t, os.Chmod(filepath.Dir(leftover), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(leftover), 0o755) })
	h.start()
	h.enqueue(h.task("node-1", wfv1.Template{}, `true`))

	require.ErrorContains(t, h.startWorker(time.Minute).wait(), "failed to clear the pool")
	_, claimed := h.fake.Result(workerNamespace, "node-1")
	assert.False(t, claimed)

	require.NoError(t, os.Chmod(filepath.Dir(leftover), 0o755))
	run := h.startWorker(time.Minute)
	assert.Equal(t, wfv1.NodeSucceeded, h.completed("node-1").Phase)
	assert.NoFileExists(t, leftover)
	require.NoError(t, run.terminate())
}

// A task main fails, and one main dies running, fail as Failed, as a pod
// whose main container fails; the worker survives main's restart and runs
// the next task.
func TestWorker_MainsFailureFailsTheTask(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.start()
	run := h.startWorker(time.Minute)

	h.enqueue(h.task("exit", wfv1.Template{}, `exit 3`))
	r := h.completed("exit")
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Equal(t, "main: Error (exit code 3)", r.Message)
	require.NotNil(t, r.Outputs)
	require.NotNil(t, r.Outputs.ExitCode)
	assert.Equal(t, "3", *r.Outputs.ExitCode)

	h.enqueue(h.task("killed", wfv1.Template{}, `echo $$ > "$OUT/pid"; exec sleep 300`))
	h.waitFile(filepath.Join(h.out, "pid"))
	orphan := readPid(t, filepath.Join(h.out, "pid"))
	t.Cleanup(func() { _ = syscall.Kill(orphan, syscall.SIGKILL) })
	h.signal(syscall.SIGKILL)
	require.Error(t, h.waitExit(loopWait))
	// In a pod, the container's processes die with it.
	require.NoError(t, syscall.Kill(orphan, syscall.SIGKILL))
	r = h.completed("killed")
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Equal(t, "main: the container exited before the task ended", r.Message)

	h.start()
	h.enqueue(h.task("next", wfv1.Template{}, `true`))
	r = h.completed("next")
	assert.Equal(t, wfv1.NodeSucceeded, r.Phase, r.Message)
	require.NoError(t, run.terminate())
	h.assertPoolClean()
}

// A task the worker fails to prepare, here as its input artifact isn't
// found, is never offered to main and fails as Error, as a pod whose init
// container fails; the worker runs the next task.
func TestWorker_PrepareFailureErrorsTheTask(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	srv := newArtifactServer(t, nil)
	h.start()
	run := h.startWorker(time.Minute)

	h.enqueue(h.task("missing", wfv1.Template{Inputs: wfv1.Inputs{Artifacts: wfv1.Artifacts{{
		Name: "in", Path: filepath.Join(h.image, "in"), ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/missing"}},
	}}}}, `touch "$OUT/ran"`))
	r := h.completed("missing")
	assert.Equal(t, wfv1.NodeError, r.Phase)
	assert.Contains(t, r.Message, "artifact in failed to load")

	h.enqueue(h.task("next", wfv1.Template{}, `true`))
	r = h.completed("next")
	assert.Equal(t, wfv1.NodeSucceeded, r.Phase, r.Message)
	require.NoError(t, run.terminate())
	assert.NoFileExists(t, filepath.Join(h.out, "ran"), "main never ran it")
	h.assertPoolClean()
}

// A worker that restarted while main ran its task kills the task and fails
// its claim as Error before taking work, and never runs the task again.
func TestWorker_RestartFailsTheTaskItHad(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.start()
	// What the worker's last run left: a claim, and the task running in
	// main.
	h.enqueue(h.task("node-1", wfv1.Template{}, ``))
	h.claim()
	task := h.offer("node-1-before", h.spec("node-1", nil, `echo run >> "$OUT/runs"; exec sleep 300`), nil)
	poolpodtest.WaitRunning(t, task, loopWait)

	run := h.startWorker(time.Minute)

	r := h.completed("node-1")
	assert.Equal(t, wfv1.NodeError, r.Phase)
	assert.Equal(t, "worker container of pod pool-pod restarted", r.Message)
	// Stopped before the claim failed, and its directory removed once main
	// was done with it.
	h.waitLog("task done", task.Dir(), "exitCode=143")
	require.Eventually(t, func() bool { return !dirExists(task.Dir()) }, loopWait, 20*time.Millisecond)
	h.enqueue(h.task("node-2", wfv1.Template{}, `true`))
	assert.Equal(t, wfv1.NodeSucceeded, h.completed("node-2").Phase)
	require.NoError(t, run.terminate())
	runs, err := os.ReadFile(filepath.Join(h.out, "runs"))
	require.NoError(t, err)
	assert.Equal(t, "run\n", string(runs), "the task isn't run again")
	r, _ = h.fake.Result(workerNamespace, "node-1")
	assert.Equal(t, wfv1.NodeError, r.Phase)
}

// A worker that restarted after offering its task but before main took it
// withdraws the task, so main never starts it, and fails its claim as
// Error.
func TestWorker_RestartWithdrawsATaskMainHasNotTaken(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.enqueue(h.task("node-1", wfv1.Template{}, ``))
	h.claim()
	task := h.offer("node-1-before", h.spec("node-1", nil, `touch "$OUT/ran"`), nil)

	run := h.startWorker(time.Minute)
	r := h.completed("node-1")
	assert.Equal(t, wfv1.NodeError, r.Phase)
	assert.Equal(t, "worker container of pod pool-pod restarted", r.Message)

	// The worker removes the withdrawn task's directory before main starts,
	// so main finds nothing offered.
	require.Eventually(t, func() bool { return !dirExists(task.Dir()) }, loopWait, 20*time.Millisecond)
	h.start()
	h.waitLog("skipping task", task.Dir(), poolpod.ErrNotOffered.Error())
	h.enqueue(h.task("node-2", wfv1.Template{}, `true`))
	assert.Equal(t, wfv1.NodeSucceeded, h.completed("node-2").Phase)
	require.NoError(t, run.terminate())
	assert.NoFileExists(t, filepath.Join(h.out, "ran"))
}

// A worker that can't report its task's result exits, leaving the claim,
// which its restart fails as Error without running the task again.
func TestWorker_ReportingGivesUp(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.start()
	h.fake.SetFault(func(op dispatch.Op) error {
		if op.Resource == dispatch.ResourceResults && op.Verb == "patch" {
			return apierr.NewServiceUnavailable("the API server is down")
		}
		return nil
	})
	h.enqueue(h.task("node-1", wfv1.Template{}, `echo run >> "$OUT/runs"`))

	err := h.startWorker(500 * time.Millisecond).wait()
	require.ErrorContains(t, err, "failed to report the task's result")
	r, ok := h.fake.Result(workerNamespace, "node-1")
	require.True(t, ok)
	assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted], "the claim is left")

	h.fake.SetFault(nil)
	run := h.startWorker(time.Minute)
	r = h.completed("node-1")
	assert.Equal(t, wfv1.NodeError, r.Phase)
	assert.Equal(t, "worker container of pod pool-pod restarted", r.Message)
	require.NoError(t, run.terminate())
	runs, err := os.ReadFile(filepath.Join(h.out, "runs"))
	require.NoError(t, err)
	assert.Equal(t, "run\n", string(runs))
}

// Each task starts with every emptyDir volume empty, of the template's own
// and of the workflow's, and no other task's directories left, even those
// an earlier run of the worker left; other volumes keep what tasks write.
func TestWorker_EachTaskStartsClean(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	vol := func(name string) string { return filepath.Join(h.root, "vol", name) }
	for _, name := range []string{"empty", "wf-empty", "keep"} {
		require.NoError(t, os.MkdirAll(vol(name), 0o755))
	}
	pool := filepath.Join(h.base.VarRunArgo, poolpod.DirName)
	for _, f := range []string{
		filepath.Join(pool, "stale", "taken"),
		filepath.Join(h.base.InputArtifacts, "stale", "in"),
		filepath.Join(h.base.Staging, "stale", "script"),
		filepath.Join(h.base.Scratch, "outputs", "stale", "outputs", "logs", "main.log"),
		filepath.Join(vol("empty"), "stale"),
		filepath.Join(vol("wf-empty"), "stale"),
	} {
		require.NoError(t, os.MkdirAll(filepath.Dir(f), 0o755))
		require.NoError(t, os.WriteFile(f, nil, 0o644))
	}
	emptyDir := apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}
	tmpl := wfv1.Template{
		Volumes: []apiv1.Volume{
			{Name: "empty", VolumeSource: emptyDir},
			{Name: "keep", VolumeSource: apiv1.VolumeSource{HostPath: &apiv1.HostPathVolumeSource{Path: vol("keep")}}},
		},
		Script: &wfv1.ScriptTemplate{Container: apiv1.Container{VolumeMounts: []apiv1.VolumeMount{
			{Name: "empty", MountPath: vol("empty")},
			{Name: "wf-empty", MountPath: vol("wf-empty")},
			{Name: "keep", MountPath: vol("keep")},
		}}},
	}
	// The task's directories are the only ones left beneath the pool pod's
	// shared ones, and only the protocol's own files besides them in the
	// pool directory.
	clean := `
test -z "$(ls -A "` + vol("empty") + `")"
test -z "$(ls -A "` + vol("wf-empty") + `")"
own="$(basename "$(dirname "$0")")"
for dir in "` + h.base.InputArtifacts + `" "` + h.base.Staging + `" "` + filepath.Join(h.base.Scratch, "outputs") + `"; do
	test "$(ls -A "$dir")" = "$own"
done
test "$(ls -A "` + pool + `" | grep -v -x -e "$own" -e ` + poolpod.CurrentFile + ` -e ` + poolpod.MainFile + `)" = ""
`
	task := func(nodeID, script string) dispatch.Task {
		task := h.task(nodeID, tmpl, clean+script)
		task.Pool.WorkflowSpec = &wfv1.WorkflowSpec{Volumes: []apiv1.Volume{{Name: "wf-empty", VolumeSource: emptyDir}}}
		return task
	}
	h.start()
	run := h.startWorker(time.Minute)

	h.enqueue(task("one", `touch "`+vol("empty")+`/one" "`+vol("wf-empty")+`/one" "`+vol("keep")+`/one"`))
	r := h.completed("one")
	require.Equal(t, wfv1.NodeSucceeded, r.Phase, "the first task started clean: %s", r.Message)
	h.enqueue(task("two", `test -e "`+vol("keep")+`/one"`))
	r = h.completed("two")
	require.Equal(t, wfv1.NodeSucceeded, r.Phase, "the second task started clean: %s", r.Message)

	require.NoError(t, run.terminate())
	h.assertPoolClean()
	for _, name := range []string{"empty", "wf-empty"} {
		entries, err := os.ReadDir(vol(name))
		require.NoError(t, err)
		assert.Empty(t, entries, "%s is wiped after the task too", name)
	}
	assert.FileExists(t, filepath.Join(vol("keep"), "one"))
}

// Each task gets directories of its own, even a node's task claimed again,
// since main runs the task current names once current changes: here the
// staging directory its script is in.
func TestWorker_TaskDirectoriesAreUnique(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.start()
	run := h.startWorker(time.Minute)
	h.enqueue(h.task("node-1", wfv1.Template{}, `basename "$(dirname "$0")" >> "$OUT/dirs"`))
	h.completed("node-1")
	// Released: the claim is deleted, and the task claimed again.
	require.True(t, h.fake.DeleteResult(workerNamespace, "node-1"))
	h.completed("node-1")
	require.NoError(t, run.terminate())
	dirs, err := os.ReadFile(filepath.Join(h.out, "dirs"))
	require.NoError(t, err)
	names := strings.Fields(string(dirs))
	require.Len(t, names, 2)
	assert.NotEqual(t, names[0], names[1])
	for _, name := range names {
		assert.True(t, strings.HasPrefix(name, "node-1-"), name)
	}
}
