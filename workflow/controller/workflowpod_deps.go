package controller

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/entrypoint"

	"go.opentelemetry.io/otel/trace"
)

// This file adapts *wfOperationCtx to the podBuilderDeps interface consumed by
// podBuilder.build. Each read-only method is a thin wrapper around woc /
// controller behaviour so the pure builder never holds a concrete
// *wfOperationCtx / *WorkflowController reference. The side-effecting wrappers
// here (createConfigMap, createPod, reserveRateLimiter, incrementActivePods,
// setNodeProgress, markNodeFailedOnShutdown, getPod, startCreateWorkflowPodSpan)
// are NOT part of podBuilderDeps; they are called by the submit core
// (woc.createPodFromBuild / woc.submitPod, workflowpod_submit.go) and by the
// dispatch layer woc.createWorkflowPod.

// compile-time assertion that *wfOperationCtx satisfies podBuilderDeps.
var _ podBuilderDeps = (*wfOperationCtx)(nil)

// lookupImage performs the read-only entrypoint/image lookup used for emissary
// command injection.
func (woc *wfOperationCtx) lookupImage(ctx context.Context, image string, opts entrypoint.Options) (*entrypoint.Image, error) {
	return woc.controller.entrypoint.Lookup(ctx, image, opts)
}

// getNodeByName returns the node with the given name from live workflow status.
func (woc *wfOperationCtx) getNodeByName(nodeName string) (*wfv1.NodeStatus, error) {
	return woc.wf.GetNodeByName(nodeName)
}

// findRetryNode locates the closest retry-node ancestor of nodeID in live status.
func (woc *wfOperationCtx) findRetryNode(nodeID string) *wfv1.NodeStatus {
	return FindRetryNode(woc.wf.Status.Nodes, nodeID)
}

// retryStrategyForTemplate resolves the retry strategy applicable to tmpl.
func (woc *wfOperationCtx) retryStrategyForTemplate(tmpl *wfv1.Template) *wfv1.RetryStrategy {
	return woc.retryStrategy(tmpl)
}

// retryNodeTemplate resolves the template for a retry node, falling back to the
// supplied template when the retry node carries no template reference.
func (woc *wfOperationCtx) retryNodeTemplate(ctx context.Context, retryNode *wfv1.NodeStatus, fallback *wfv1.Template) (*wfv1.Template, error) {
	if retryNode.TemplateName == "" && retryNode.TemplateRef == nil {
		return fallback, nil
	}
	scope, name := retryNode.GetTemplateScope()
	tmplCtx, err := woc.createTemplateContext(ctx, scope, name)
	if err != nil {
		return nil, err
	}
	_, resolvedTmpl, _, err := tmplCtx.ResolveTemplate(ctx, retryNode)
	if err != nil {
		return nil, err
	}
	return resolvedTmpl, nil
}

// applyRetryOnDifferentHost applies the retry-on-different-host affinity tweak
// using live workflow status.
func (woc *wfOperationCtx) applyRetryOnDifferentHost(retryNodeID string, retryStrategy wfv1.RetryStrategy, pod *apiv1.Pod) {
	RetryOnDifferentHost(retryNodeID)(retryStrategy, woc.wf.Status.Nodes, pod)
}

// persistentVolumeClaims returns the workflow's live PVC volume references.
func (woc *wfOperationCtx) persistentVolumeClaims() []apiv1.Volume {
	return woc.wf.Status.PersistentVolumeClaims
}

// markNodeFailedOnShutdown marks the node failed because the workflow is
// shutting down. Side effect — called by woc.createWorkflowPod.
func (woc *wfOperationCtx) markNodeFailedOnShutdown(ctx context.Context, nodeName, message string) {
	woc.markNodePhase(ctx, nodeName, wfv1.NodeFailed, message)
}

// setNodeProgress records initial progress parsed from pod metadata onto the
// node status. Side effect — called by woc.submitPod.
func (woc *wfOperationCtx) setNodeProgress(ctx context.Context, nodeID string, progress wfv1.Progress) {
	node, getNodeErr := woc.wf.Status.Nodes.Get(nodeID)
	if getNodeErr != nil {
		logging.RequireLoggerFromContext(ctx).WithPanic().Error(ctx, "was unable to obtain node")
	}
	node.Progress = progress
	woc.wf.Status.Nodes.Set(ctx, nodeID, *node)
}

// startCreateWorkflowPodSpan opens the create_workflow_pod tracing span.
func (woc *wfOperationCtx) startCreateWorkflowPodSpan(ctx context.Context, nodeID string) (context.Context, trace.Span) {
	return woc.controller.tracing.StartCreateWorkflowPod(ctx, nodeID)
}

// createConfigMap creates the env/args offload ConfigMap. Side effect — called
// by woc.createPodFromBuild (build returns the object as an ExtraObject instead).
func (woc *wfOperationCtx) createConfigMap(ctx context.Context, cm *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	return woc.controller.kubeclientset.CoreV1().ConfigMaps(woc.wf.ObjectMeta.Namespace).Create(ctx, cm, metav1.CreateOptions{})
}

// reserveRateLimiter reserves a resource-creation slot from the controller rate
// limiter, recording latency. Side effect — called by woc.createPodFromBuild.
func (woc *wfOperationCtx) reserveRateLimiter(ctx context.Context) error {
	reservation := woc.controller.rateLimiter.Reserve()
	if !reservation.OK() {
		reservation.Cancel()
		return ErrResourceRateLimitReached
	}
	delay := reservation.Delay()
	woc.controller.metrics.RecordResourceRateLimiterLatency(ctx, delay.Seconds())
	if delay > 0 {
		reservation.Cancel()
		return ErrResourceRateLimitReached
	}
	return nil
}

// createPod creates the workflow pod. Side effect — called by woc.createPodFromBuild.
func (woc *wfOperationCtx) createPod(ctx context.Context, pod *apiv1.Pod) (*apiv1.Pod, error) {
	return woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
}

// getPod fetches an existing pod by name (AlreadyExists recovery). Side effect —
// called by woc.createPodFromBuild.
func (woc *wfOperationCtx) getPod(ctx context.Context, name string) (*apiv1.Pod, error) {
	return woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Get(ctx, name, metav1.GetOptions{})
}

// incrementActivePods bumps the active-pod parallelism counter. Side effect —
// called by woc.submitPod.
func (woc *wfOperationCtx) incrementActivePods() {
	woc.activePods++
}
