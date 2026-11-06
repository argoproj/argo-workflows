package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// These tests stop a pool pod's tasks from outside main, as the kubelet and
// the controller stop a pod's: deadlines, interrupts, lost claims, and the
// worker's SIGTERM.

// released waits for the task's claim to be deleted, as releasing it does,
// while its TaskSet stays for another worker to claim.
func (h *workerHarness) released(nodeID string) {
	h.t.Helper()
	require.Eventually(h.t, func() bool {
		_, ok := h.fake.Result(workerNamespace, nodeID)
		return !ok
	}, loopWait, 20*time.Millisecond, "%s was never released", nodeID)
	_, ok := h.fake.TaskSet(workerNamespace, dispatch.Key{NodeID: nodeID}.TaskSetName())
	assert.True(h.t, ok, "a released task can be claimed again")
}

// currentTask is the task main runs next, once the worker has offered one.
func (h *workerHarness) currentTask() poolpod.Task {
	h.t.Helper()
	pool := poolpod.NewPool(filepath.Join(h.base.VarRunArgo, poolpod.DirName))
	var task poolpod.Task
	require.Eventually(h.t, func() bool {
		t, ok, err := pool.Current()
		task = t
		return err == nil && ok
	}, loopWait, 10*time.Millisecond, "no task was offered")
	return task
}

// On SIGTERM, a task that ends before the release is due completes, and
// one main is still running then is stopped and released before the pod's
// grace period ends, without being reported.
func TestWorker_SIGTERM(t *testing.T) {
	t.Parallel()
	t.Run("ShortTaskCompletes", func(t *testing.T) {
		t.Parallel()
		h := newWorkerHarness(t)
		// The task ends well before the release is due, however late the
		// worker sees it end.
		h.grace, h.reserve = 7*time.Second, 2*time.Second
		h.start()
		run := h.startWorker(time.Minute)
		h.enqueue(h.task("short", wfv1.Template{}, `touch "$OUT/started"; sleep 0.5; touch "$OUT/finished"`))
		h.waitFile(filepath.Join(h.out, "started"))

		require.NoError(t, run.terminate())

		r, ok := h.fake.Result(workerNamespace, "short")
		require.True(t, ok)
		assert.Equal(t, wfv1.NodeSucceeded, r.Phase, r.Message)
		assert.FileExists(t, filepath.Join(h.out, "finished"))
	})
	t.Run("LongTaskIsReleased", func(t *testing.T) {
		t.Parallel()
		h := newWorkerHarness(t)
		h.grace, h.reserve = 5*time.Second, 3*time.Second
		h.start()
		run := h.startWorker(time.Minute)
		h.enqueue(h.task("long", wfv1.Template{}, `touch "$OUT/started"; exec sleep 300`))
		h.waitFile(filepath.Join(h.out, "started"))
		task := h.currentTask()

		terminated := time.Now()
		require.NoError(t, run.terminate())
		elapsed := time.Since(terminated)

		assert.GreaterOrEqual(t, elapsed, h.grace-h.reserve, "the task had until the release was due")
		assert.Less(t, elapsed, h.grace, "released before the kubelet's SIGKILL")
		h.released("long")
		assert.NoDirExists(t, task.Dir(), "main was done with it before it was removed")
		h.assertPoolClean()
	})
}

// A task past its activeDeadlineSeconds, counted from the claim, or past its
// absolute deadline is killed and fails as a pod's does: with the kubelet's
// message for the one, which it enforces for a pod, and the executor's for
// the other. The worker then takes the next task.
func TestWorker_Deadline(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		set     func(*wfv1.TaskDispatch)
		message string
	}{
		// The claim's time has the API server's one-second precision, so
		// this is at least a second after the claim.
		"ActiveDeadlineSeconds": {
			set:     func(d *wfv1.TaskDispatch) { d.ActiveDeadlineSeconds = ptr.To[int64](2) },
			message: "Pod was active on the node longer than the specified deadline",
		},
		// Set as the task is enqueued, once the worker has run a task, so
		// the deadline leaves time for the task to be claimed, prepared and
		// taken by main before it passes.
		"Absolute": {
			set: func(d *wfv1.TaskDispatch) {
				d.Deadline = &metav1.Time{Time: time.Now().Add(3 * time.Second)}
				d.ActiveDeadlineSeconds = ptr.To[int64](3600)
			},
			message: "Step exceeded its deadline",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newWorkerHarness(t)
			h.start()
			run := h.startWorker(time.Minute)
			// The worker and main are up before the deadline starts.
			h.enqueue(h.task("first", wfv1.Template{}, `true`))
			require.Equal(t, wfv1.NodeSucceeded, h.completed("first").Phase)
			task := h.task("late", wfv1.Template{}, `exec sleep 300`)
			tt.set(&task.Dispatch)
			h.enqueue(task)

			r := h.completed("late")
			assert.Equal(t, wfv1.NodeFailed, r.Phase)
			assert.Equal(t, tt.message, r.Message)
			require.NotNil(t, r.Outputs)
			require.NotNil(t, r.Outputs.ExitCode)
			assert.Equal(t, "143", *r.Outputs.ExitCode)

			h.enqueue(h.task("next", wfv1.Template{}, `true`))
			assert.Equal(t, wfv1.NodeSucceeded, h.completed("next").Phase)
			require.NoError(t, run.terminate())
		})
	}
	t.Run("PassedWhilePreparing", func(t *testing.T) {
		t.Parallel()
		h := newWorkerHarness(t)
		h.start()
		srv := newBlockingServer(t)
		run := h.startWorker(time.Minute)
		task := h.task("late", wfv1.Template{Inputs: wfv1.Inputs{Artifacts: wfv1.Artifacts{{
			Name: "in", Path: filepath.Join(h.image, "in"), ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/in"}},
		}}}}, `touch "$OUT/ran"`)
		task.Dispatch.ActiveDeadlineSeconds = ptr.To[int64](2)
		enqueued := time.Now()
		h.enqueue(task)

		r := h.completed("late")
		assert.GreaterOrEqual(t, time.Since(enqueued), time.Second, "stopped while its input was downloading")
		assert.Equal(t, wfv1.NodeFailed, r.Phase)
		assert.Equal(t, "Pod was active on the node longer than the specified deadline", r.Message, "a pod's activeDeadlineSeconds counts its init time too")
		require.NoError(t, run.terminate())
		assert.NoFileExists(t, filepath.Join(h.out, "ran"), "main never ran it")
	})
}

// Labelling a running task's claim interrupted stops the task within
// seconds; the worker completes the claim with nothing the task produced,
// and takes the next task.
func TestWorker_Interrupt(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.start()
	run := h.startWorker(time.Minute)
	out := filepath.Join(h.image, "out")
	h.enqueue(h.task("node-1", wfv1.Template{
		Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: out}}}},
	}, `echo produced > "`+out+`"; touch "$OUT/started"; exec sleep 300`))
	h.waitFile(filepath.Join(h.out, "started"))
	r, ok := h.fake.Result(workerNamespace, "node-1")
	require.True(t, ok)
	ts, ok := h.fake.TaskSet(workerNamespace, dispatch.Key{NodeID: "node-1"}.TaskSetName())
	require.True(t, ok)

	interrupted := time.Now()
	require.NoError(t, h.fake.Dispatcher().Interrupt(logging.TestContext(t.Context()), dispatch.Key{Namespace: workerNamespace, NodeID: "node-1"}, ts.UID, r.UID))

	r = h.completed("node-1")
	assert.Less(t, time.Since(interrupted), 5*time.Second)
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Equal(t, "the task was interrupted", r.Message)
	assert.Nil(t, r.Outputs, "nothing the task produced")
	h.enqueue(h.task("next", wfv1.Template{}, `true`))
	assert.Equal(t, wfv1.NodeSucceeded, h.completed("next").Phase)
	require.NoError(t, run.terminate())
	h.assertPoolClean()
}

// A running task whose claim is deleted, as when its workflow is deleted or
// the controller takes the claim for a release, is killed within seconds and
// nothing is reported for it.
func TestWorker_LostClaim(t *testing.T) {
	t.Parallel()
	for name, deleteTaskSet := range map[string]bool{"WorkflowDeleted": true, "ClaimDeleted": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newWorkerHarness(t)
			h.start()
			run := h.startWorker(time.Minute)
			h.enqueue(h.task("node-1", wfv1.Template{}, `
echo run >> "$OUT/runs"
if [ "$(wc -l < "$OUT/runs")" -eq 1 ]; then echo $$ > "$OUT/pid"; exec sleep 300; fi`))
			h.waitFile(filepath.Join(h.out, "pid"))
			pid := readPid(t, filepath.Join(h.out, "pid"))
			t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
			first, ok := h.fake.Result(workerNamespace, "node-1")
			require.True(t, ok)

			lost := time.Now()
			if deleteTaskSet {
				require.True(t, h.fake.DeleteTaskSet(workerNamespace, dispatch.Key{NodeID: "node-1"}.TaskSetName()))
			}
			require.True(t, h.fake.DeleteResult(workerNamespace, "node-1"))

			require.Eventually(t, func() bool { return gone(pid) }, 5*time.Second, 20*time.Millisecond, "the task wasn't killed")
			assert.Less(t, time.Since(lost), 5*time.Second)
			if deleteTaskSet {
				h.enqueue(h.task("next", wfv1.Template{}, `true`))
				assert.Equal(t, wfv1.NodeSucceeded, h.completed("next").Phase)
				_, ok = h.fake.Result(workerNamespace, "node-1")
				assert.False(t, ok, "nothing is reported, and the claim isn't recreated")
			} else {
				// The TaskSet is claimable again, and runs afresh.
				r := h.completed("node-1")
				assert.NotEqual(t, first.UID, r.UID, "the lost claim isn't the one completed")
				assert.Equal(t, wfv1.NodeSucceeded, r.Phase, r.Message)
			}
			require.NoError(t, run.terminate())
		})
	}
}

// SIGTERM after main has finished the task, while its outputs are being
// uploaded, completes the task rather than releasing it, even once the
// release would be due.
func TestWorker_SIGTERMWhileCollecting(t *testing.T) {
	t.Parallel()
	h := newWorkerHarness(t)
	h.grace, h.reserve = 3*time.Second, 2*time.Second
	uploading := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		uploading <- struct{}{}
		// Past the release.
		time.Sleep(h.grace - h.reserve + time.Second)
	}))
	t.Cleanup(srv.Close)
	out := filepath.Join(h.image, "out")
	h.start()
	run := h.startWorker(time.Minute)
	h.enqueue(h.task("node-1", wfv1.Template{Outputs: wfv1.Outputs{Artifacts: wfv1.Artifacts{{
		Name: "a", Path: out, ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/a"}},
	}}}}, `echo artifact > "`+out+`"`))
	<-uploading

	require.NoError(t, run.terminate())

	r, ok := h.fake.Result(workerNamespace, "node-1")
	require.True(t, ok)
	assert.Equal(t, wfv1.NodeSucceeded, r.Phase, r.Message)
	require.NotNil(t, r.Outputs)
	require.Len(t, r.Outputs.Artifacts, 1)
}

// blockingServer is an artifact server whose downloads never finish.
type blockingServer struct {
	*httptest.Server
	// downloading has a value once a download has started.
	downloading chan struct{}
}

func newBlockingServer(t *testing.T) *blockingServer {
	t.Helper()
	s := &blockingServer{downloading: make(chan struct{}, 1)}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "partial")
		w.(http.Flusher).Flush()
		select {
		case s.downloading <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	t.Cleanup(s.Close)
	return s
}

// waitDownloading waits for a download from s to start.
func (s *blockingServer) waitDownloading(t *testing.T) {
	t.Helper()
	select {
	case <-s.downloading:
	case <-time.After(loopWait):
		require.FailNow(t, "the worker never started downloading")
	}
}

// SIGTERM while the worker stages a task's inputs, or once it has offered
// the task but before main takes it, releases the task at once, and main
// never starts it.
func TestWorker_SIGTERMBeforeMainTakesTheTask(t *testing.T) {
	t.Parallel()
	t.Run("WhileStaging", func(t *testing.T) {
		t.Parallel()
		h := newWorkerHarness(t)
		srv := newBlockingServer(t)
		h.start()
		run := h.startWorker(time.Minute)
		h.enqueue(h.task("node-1", wfv1.Template{Inputs: wfv1.Inputs{Artifacts: wfv1.Artifacts{{
			Name: "in", Path: filepath.Join(h.image, "in"), ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: srv.URL + "/in"}},
		}}}}, `touch "$OUT/ran"`))
		srv.waitDownloading(t)

		terminated := time.Now()
		require.NoError(t, run.terminate())

		assert.Less(t, time.Since(terminated), 5*time.Second, "released at once")
		h.released("node-1")
		assert.NoFileExists(t, filepath.Join(h.out, "ran"))
		h.assertPoolClean()
	})
	t.Run("AfterTheOffer", func(t *testing.T) {
		t.Parallel()
		h := newWorkerHarness(t)
		// main is up as far as the worker can tell, but isn't running, so
		// it can't take the task.
		require.NoError(t, h.pool.Init())
		mainLock, err := h.pool.HoldMain(t.Context())
		require.NoError(t, err)
		run := h.startWorker(time.Minute)
		h.enqueue(h.task("node-1", wfv1.Template{}, `touch "$OUT/ran"`))
		task := h.currentTask()

		terminated := time.Now()
		require.NoError(t, run.terminate())

		assert.Less(t, time.Since(terminated), 5*time.Second, "released at once")
		h.released("node-1")
		require.NoError(t, mainLock.Close())
		h.start()
		h.waitLog("skipping task", task.Dir())
		assert.NoFileExists(t, filepath.Join(h.out, "ran"))
	})
}
