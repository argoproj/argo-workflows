package pooled

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// claimed claims task on a fake and returns the claim.
func claimed(t *testing.T, task dispatch.Task) dispatch.Claimed {
	t.Helper()
	f := dispatch.NewFake()
	enqueue(t, f, task)
	ctx := logging.TestContext(t.Context())
	c := f.Consumer(testWorker("worker-a"))
	candidates, err := c.Candidates(ctx)
	require.NoError(t, err)
	h, err := c.Claim(ctx, candidates[0])
	require.NoError(t, err)
	return h
}

func TestTaskConfig(t *testing.T) {
	claimedAt := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	task := testTask("node-1")
	task.Template.Inputs = wfv1.Inputs{
		Parameters: []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("v")}},
		Artifacts:  wfv1.Artifacts{{Name: "a", Path: "/a"}},
	}
	task.Dispatch = wfv1.TaskDispatch{
		TaskName:                  "my-wf-main-node-1",
		IncludeScriptOutput:       true,
		InputArtifactPlugins:      []wfv1.ArtifactPluginName{"in"},
		Command:                   []string{"/entrypoint"},
		Deadline:                  &metav1.Time{Time: claimedAt.Add(time.Hour).Local()},
		TraceContext:              map[string]string{"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
		ProgressPatchTickDuration: &metav1.Duration{Duration: time.Minute},
		ProgressFileTickDuration:  &metav1.Duration{Duration: 3 * time.Second},
	}
	h := claimed(t, task)
	claim := h.Claim()
	claim.CreatedAt = metav1.Time{Time: claimedAt}

	cfg, err := TaskConfig(h.TaskSet(), claim)
	require.NoError(t, err)
	assert.Equal(t, executor.TaskConfig{
		WorkflowName:        "my-wf",
		WorkflowUID:         "wf-uid",
		NodeID:              "node-1",
		Template:            *common.ExecutorTemplate(&task.Template),
		IncludeScriptOutput: true,
		Deadline:            claimedAt.Add(time.Hour),
		ProgressFile:        common.ArgoProgressPath,
		// As a pod's.
		ProgressPatchTickDuration: time.Minute,
		ProgressFileTickDuration:  3 * time.Second,
		TraceContext:              map[string]string{"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
		InputArtifactPluginNames:  []wfv1.ArtifactPluginName{"in"},
		Command:                   []string{"/entrypoint"},
	}, cfg)
	assert.Empty(t, cfg.Template.Inputs.Parameters, "substituted already, as a pod's executor gets it")

	t.Run("ActiveDeadlineSeconds", func(t *testing.T) {
		ts := h.TaskSet().DeepCopy()
		ts.Spec.Dispatch.ActiveDeadlineSeconds = ptr.To[int64](60)
		cfg, err := TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.Equal(t, claimedAt.Add(time.Minute), cfg.Deadline, "counted from the claim, when earlier")
		assert.Equal(t, cfg.Deadline, cfg.ActiveDeadline, "the deadline is activeDeadlineSeconds'")
		ts.Spec.Dispatch.ActiveDeadlineSeconds = ptr.To[int64](7200)
		cfg, err = TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.Equal(t, claimedAt.Add(time.Hour), cfg.Deadline)
		assert.True(t, cfg.ActiveDeadline.IsZero(), "the absolute deadline is earlier")
		ts.Spec.Dispatch.ActiveDeadlineSeconds = ptr.To[int64](3600)
		cfg, err = TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.Equal(t, claimedAt.Add(time.Hour), cfg.ActiveDeadline, "activeDeadlineSeconds wins a tie, as the kubelet's message does for a pod")
		ts.Spec.Dispatch.ActiveDeadlineSeconds = ptr.To[int64](7200)
		ts.Spec.Dispatch.Deadline = nil
		cfg, err = TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.Equal(t, claimedAt.Add(2*time.Hour), cfg.Deadline)
		assert.Equal(t, cfg.Deadline, cfg.ActiveDeadline)
	})
	t.Run("NoDeadline", func(t *testing.T) {
		ts := h.TaskSet().DeepCopy()
		ts.Spec.Dispatch.Deadline = nil
		cfg, err := TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.True(t, cfg.Deadline.IsZero())
	})
	t.Run("NoProgress", func(t *testing.T) {
		ts := h.TaskSet().DeepCopy()
		ts.Spec.Dispatch.ProgressFileTickDuration = nil
		cfg, err := TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.Empty(t, cfg.ProgressFile)
		assert.Zero(t, cfg.ProgressPatchTickDuration)
	})
	t.Run("EmptyDirs", func(t *testing.T) {
		ts := h.TaskSet().DeepCopy()
		tmpl := ts.Spec.Tasks["node-1"]
		emptyDir := apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}
		claimVolume := apiv1.VolumeSource{PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{ClaimName: "c"}}
		tmpl.Volumes = []apiv1.Volume{{Name: "own", VolumeSource: emptyDir}, {Name: "shadowed", VolumeSource: claimVolume}}
		tmpl.Container.VolumeMounts = []apiv1.VolumeMount{
			{Name: "own", MountPath: "/own"},
			{Name: "shadowed", MountPath: "/shadowed"},
			{Name: "wf-empty", MountPath: "/wf-empty"},
			{Name: "wf-claim", MountPath: "/wf-claim"},
			{Name: "own", MountPath: "/own-again", SubPath: "sub"},
		}
		ts.Spec.Tasks["node-1"] = tmpl
		ts.Spec.Pool.WorkflowSpec = &wfv1.WorkflowSpec{Volumes: []apiv1.Volume{
			{Name: "wf-empty", VolumeSource: emptyDir},
			{Name: "wf-claim", VolumeSource: claimVolume},
			{Name: "shadowed", VolumeSource: emptyDir},
		}}
		cfg, err := TaskConfig(ts, claim)
		require.NoError(t, err)
		assert.Equal(t, []string{"/own", "/wf-empty", "/own-again"}, cfg.EmptyDirs, "the template's own volume wins over the workflow's of the same name")
	})
	t.Run("NoDispatch", func(t *testing.T) {
		ts := h.TaskSet().DeepCopy()
		ts.Spec.Dispatch = nil
		_, err := TaskConfig(ts, claim)
		require.ErrorIs(t, err, dispatch.ErrFieldsDropped)
	})
	t.Run("OtherTask", func(t *testing.T) {
		other := claim
		other.NodeID = "node-2"
		_, err := TaskConfig(h.TaskSet(), other)
		require.Error(t, err)
	})
}
