package controller

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/argoproj/argo-workflows/v4/errors"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// submitPod performs all the impure operations that the pure podBuilder.build
// deliberately avoids. It is the single side-effecting half of workload pod
// creation. By the time submitPod is reached the pod is known not to exist and
// the workflow is known to be executing (createWorkflowPod checks both before
// building).
//
// The order below is correctness-critical:
//
//	c-e. shared submit core — see createPodFromBuild, which owns these steps
//	f.   activePods++          — parallelism accounting
//	g.   apply ProgressToApply — initial node-status progress write
//
// Step (g) runs only after a successful create so the workflow status never
// records progress for a pod that failed to be created.
func (woc *wfOperationCtx) submitPod(ctx context.Context, result *podBuildResult, nodeName, nodeID string, baseLog logging.Logger) (*apiv1.Pod, error) {
	log := baseLog.WithFields(logging.Fields{"nodeName": nodeName, "nodeID": nodeID})
	ctx = logging.WithLogger(ctx, log)

	pod := result.Pod

	// The create_workflow_pod span is started by createWorkflowPod BEFORE pb.build
	// (so the span covers construction and the injected W3C traceparent reflects it),
	// and is carried in ctx. Here we read it back to stamp the pod's trace/span-ID
	// annotations, which must match the traceparent injected into the pod env.
	//
	// Only set trace/span ID annotations when the span context is valid to avoid
	// writing all-zero IDs. The agent pod is not traced and skips this.
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		pod.Annotations[common.AnnotationKeyTraceID] = sc.TraceID().String()
		pod.Annotations[common.AnnotationKeySpanID] = sc.SpanID().String()
	}

	// (c-e) shared submit core: ExtraObjects, rate-limiter, k8s Create with
	// AlreadyExists recovery and transient passthrough. fresh reports whether the
	// pod was genuinely created by this call, versus recovered via Get on the
	// AlreadyExists path.
	created, fresh, err := woc.createPodFromBuild(ctx, result, log)
	if err != nil {
		return nil, err
	}

	// (f) bump active-pod parallelism accounting. This is a node-dispatch concern
	// (workload parallelism), so it lives in the workload wrapper, not the shared
	// primitive — the agent pod is not subject to workflow parallelism limits.
	//
	// Only count a genuinely fresh create. On the AlreadyExists-recovery path the
	// pod was created on a prior reconcile (and already counted then), so
	// re-incrementing here would over-count activePods and could briefly defer
	// launching another pod against Spec.Parallelism.
	if fresh {
		woc.incrementActivePods()
	}

	// (g) apply initial progress to the node status, if any was parsed from pod
	// metadata during build.
	if result.ProgressToApply != nil {
		woc.setNodeProgress(ctx, nodeID, *result.ProgressToApply)
	}

	return created, nil
}

// createPodFromBuild is the shared, node-dispatch-agnostic core of pod
// submission, consumed by BOTH the workload path (woc.submitPod) and the agent
// path (woc.createAgentPod). It bakes in NO assumptions about workflow nodes,
// parallelism, tracing, or shutdown strategy — those are the caller's concern.
//
// It performs the impure half of pod creation in a correctness-critical order:
//
//	(c) ExtraObjects (ConfigMaps FIRST) — the pod mounts them, AlreadyExists OK
//	(d) rate-limiter reserve            — throttle resource creation
//	(e) k8s Pod Create                  — with AlreadyExists recovery via Get
//
// Transient create errors are returned raw so callers' requeue logic can act
// on them; non-transient failures are wrapped as internal errors.
//
// The second return value, fresh, reports whether the pod was genuinely created
// by this call (true) versus recovered via Get on the AlreadyExists path
// (false). Callers that do parallelism accounting (woc.submitPod) must only
// count fresh creates; the agent path ignores it.
func (woc *wfOperationCtx) createPodFromBuild(ctx context.Context, result *podBuildResult, log logging.Logger) (*apiv1.Pod, bool, error) {
	pod := result.Pod

	// (c) create ExtraObjects, ConfigMaps FIRST — the pod mounts them, so they
	// must exist before the pod is created. AlreadyExists is tolerated.
	for _, cm := range result.ExtraObjects {
		created, cmErr := woc.createConfigMap(ctx, cm)
		if cmErr != nil {
			if !apierr.IsAlreadyExists(cmErr) {
				return nil, false, cmErr
			}
			log.WithField("name", cm.Name).Info(ctx, "Configmap already exists")
		} else {
			log.WithField("name", created.Name).Info(ctx, "Created configmap")
		}
	}

	// (d) reserve a resource-creation slot from the rate limiter.
	if rlErr := woc.reserveRateLimiter(ctx); rlErr != nil {
		return nil, false, rlErr
	}

	log = log.WithField("podName", pod.Name)
	ctx = logging.WithLogger(ctx, log)
	log.Debug(ctx, "Creating Pod")

	// (e) create the pod, with AlreadyExists recovery and transient-error
	// passthrough.
	created, err := woc.createPod(ctx, pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// pod names are deterministic. We can get here if the controller fails
			// to persist the workflow after creating the pod.
			log.Info(ctx, "Failed pod creation: already exists")
			// Fetch the existing pod. On success it is returned as a NON-fresh
			// recovery (fresh=false) so the caller does not re-count it for
			// parallelism. On failure the OUTER err must be reassigned to THIS
			// getPod error so the transient check below evaluates against the Get
			// failure (which may be transient and warrant a clean requeue), rather
			// than against the original AlreadyExists (never transient).
			var existing *apiv1.Pod
			if existing, err = woc.getPod(ctx, pod.Name); err == nil {
				return existing, false, nil
			}
		}
		if errorsutil.IsTransientErr(ctx, err) {
			return nil, false, err
		}
		log.WithError(err).Info(ctx, "Failed to create pod")
		return nil, false, errors.InternalWrapError(err)
	}
	log.Info(ctx, "Created pod")

	return created, true, nil
}
