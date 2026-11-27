// Package pooled is a pool pod's worker side of pooled tasks: a TaskSource
// that claims the pool's WorkflowTaskSets through workflow/dispatch and
// yields each as an executor task, built from the TaskSet as a pod's
// executor builds its task from its env.
package pooled

import (
	"fmt"
	"maps"
	"slices"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/pool"
)

// TaskConfig builds the TaskConfig of the task a claimed TaskSet holds,
// from its spec.tasks and spec.dispatch, as a pod's executor builds the
// same task's from the env its pod gets. The one difference is the
// deadline: a pod's kubelet enforces activeDeadlineSeconds, while a
// worker stops the task itself, so the deadline is the earlier of the
// task's absolute deadline and activeDeadlineSeconds after the claim,
// counted from the claim's creation as the controller counts it, and
// ActiveDeadline says when it is the latter.
func TaskConfig(taskSet *wfv1.WorkflowTaskSet, claim dispatch.Claim) (executor.TaskConfig, error) {
	task, ok := dispatch.CandidateFromTaskSet(taskSet)
	if !ok {
		return executor.TaskConfig{}, fmt.Errorf("WorkflowTaskSet %s/%s isn't a pooled task", taskSet.Namespace, taskSet.Name)
	}
	tmpl, ok := taskSet.Spec.Tasks[claim.NodeID]
	if !ok || len(taskSet.Spec.Tasks) != 1 {
		return executor.TaskConfig{}, fmt.Errorf("WorkflowTaskSet %s/%s doesn't hold exactly the task %s", taskSet.Namespace, taskSet.Name, claim.NodeID)
	}
	d := taskSet.Spec.Dispatch
	if d == nil {
		return executor.TaskConfig{}, fmt.Errorf("WorkflowTaskSet %s/%s: %w", taskSet.Namespace, taskSet.Name, dispatch.ErrFieldsDropped)
	}
	taskDeadline, activeDeadline := deadline(d, claim)
	cfg := executor.TaskConfig{
		WorkflowName:             task.WorkflowName,
		WorkflowUID:              task.WorkflowUID,
		NodeID:                   claim.NodeID,
		Template:                 *common.ExecutorTemplate(&tmpl),
		IncludeScriptOutput:      d.IncludeScriptOutput,
		Deadline:                 taskDeadline,
		ActiveDeadline:           activeDeadline,
		TraceContext:             maps.Clone(d.TraceContext),
		InputArtifactPluginNames: slices.Clone(d.InputArtifactPlugins),
		Command:                  slices.Clone(d.Command),
		EmptyDirs:                emptyDirs(&tmpl, taskSet.Spec.Pool),
	}
	// As for a pod, progress is monitored only when both ticks are set.
	if patchTick, fileTick := duration(d.ProgressPatchTickDuration), duration(d.ProgressFileTickDuration); common.ProgressMonitored(patchTick, fileTick) {
		cfg.ProgressFile = common.ArgoProgressPath
		cfg.ProgressPatchTickDuration = patchTick
		cfg.ProgressFileTickDuration = fileTick
	}
	return cfg, nil
}

// emptyDirs lists where main mounts the pool pod's emptyDir volumes: those
// of the template's own volumes and of the workflow's that it mounts, which
// spec.pool carries, resolved as a pod's mounts are (pool.MountVolume).
func emptyDirs(tmpl *wfv1.Template, poolSpec *wfv1.PoolSpec) []string {
	var volumes []apiv1.Volume
	if poolSpec != nil && poolSpec.WorkflowSpec != nil {
		volumes = poolSpec.WorkflowSpec.Volumes
	}
	var mounts []string
	for _, m := range tmpl.GetVolumeMounts() {
		if v, _ := pool.MountVolume(tmpl, volumes, m.Name); v != nil && v.EmptyDir != nil {
			mounts = append(mounts, m.MountPath)
		}
	}
	return mounts
}

// deadline is when the worker stops the task (pool.StopAt), in UTC as a
// pod parses its ARGO_DEADLINE, or zero if it has no deadline, and that
// time again as active if it is the activeDeadlineSeconds' end. active is
// zero otherwise.
func deadline(d *wfv1.TaskDispatch, claim dispatch.Claim) (deadline, active time.Time) {
	at, isActive := pool.StopAt(d.Deadline, d.ActiveDeadlineSeconds, claim.CreatedAt.Time)
	if at.IsZero() {
		return time.Time{}, time.Time{}
	}
	if isActive {
		return at.UTC(), at.UTC()
	}
	return at.UTC(), time.Time{}
}

// duration is d's duration, or zero when it isn't set.
func duration(d *metav1.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return d.Duration
}
