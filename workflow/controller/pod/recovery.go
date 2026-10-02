package pod

import (
	"context"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// ReconcilePodCleanup is a UID-bound hint, not permission to remove a barrier.
// Initial informer Add events rebuild these hints after controller restart.
func (c *Controller) ReconcilePodCleanup(ctx context.Context, pod *apiv1.Pod) {
	c.queuePodForCleanup(ctx, pod.Namespace, pod.Name, reconcilePodCleanup, string(pod.UID))
}

// QueueWorkflowCleanup handles both filter exit on completion and actual
// deletion. Its worker lists only this namespace and validates every owner.
func (c *Controller) QueueWorkflowCleanup(ctx context.Context, wf metav1.Object) {
	c.queuePodForCleanup(ctx, wf.GetNamespace(), wf.GetName(), reconcileWorkflowCleanup, string(wf.GetUID()))
}

func (c *Controller) reconcileWorkflowCleanup(ctx context.Context, namespace, name, uid string) error {
	pods, err := c.kubeclientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{common.LabelKeyWorkflow: name}.AsSelector().String(),
	})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		owner := metav1.GetControllerOf(&pod)
		if pod.Namespace == namespace && c.matchesInstance(&pod) && owner != nil && owner.Name == name && string(owner.UID) == uid && owner.Kind == workflow.WorkflowKind && owner.APIVersion == workflow.APIVersion {
			c.ReconcilePodCleanup(ctx, &pod)
		}
	}
	return nil
}

func (c *Controller) reconcilePodCleanup(ctx context.Context, pod *apiv1.Pod) error {
	state, err := c.observePodCleanup(ctx, pod, pod.DeletionTimestamp == nil || terminalPod(pod))
	if err != nil {
		return err
	}
	if state.ignored {
		return nil
	}
	if state.orphan {
		policy := podGCFromPod(pod)
		action := noAction
		if hasOurFinalizer(pod.Finalizers) {
			action = removeFinalizer
		}
		if policy.Strategy == wfv1.PodGCOnPodCompletion || policy.Strategy == wfv1.PodGCOnWorkflowCompletion || policy.Strategy == wfv1.PodGCOnPodSuccess && pod.Status.Phase == apiv1.PodSucceeded {
			action = deletePod
		}
		if action != noAction {
			delay, delayErr := policy.GetDeleteDelayDuration()
			if delayErr != nil {
				return delayErr
			}
			c.queuePodForCleanupAfter(ctx, pod.Namespace, pod.Name, action, delay, string(pod.UID))
		}
		return nil
	}
	if pod.DeletionTimestamp != nil {
		if hasOurFinalizer(pod.Finalizers) {
			// The shared mutation gate checks a terminal result again at execution.
			c.queuePodForCleanup(ctx, pod.Namespace, pod.Name, removeFinalizer, string(pod.UID))
		}
		return nil
	}
	if component := pod.Labels[common.LabelKeyComponent]; component != "" {
		if component == "agent" && state.workflow.Status.Fulfilled() && state.workflow.Labels[common.LabelKeyCompleted] == "true" {
			c.queuePodForCleanup(ctx, pod.Namespace, pod.Name, deletePod, string(pod.UID))
		}
		return nil
	}
	if restartDisposition(pod, state) {
		c.DeletePodByUID(ctx, pod.Namespace, pod.Name, string(pod.UID))
		return nil
	}
	if !terminalPod(pod) {
		if state.node != nil && state.node.Phase.Completed() && !state.node.IsDaemoned() {
			if state.node.CapturedPodUID != string(pod.UID) {
				return StatusCaptureHold(CaptureLegacyUnsupported, "stopped node has no matching pod capture provenance")
			}
			// Persisted daemon stop/cancellation can outlive its signal queue.
			c.TerminateContainers(ctx, pod.Namespace, pod.Name, string(pod.UID))
		}
		return nil
	}
	if statusCaptureEnabled() && hasOurFinalizer(pod.Finalizers) {
		if captureErr := c.ensureCapturedDisposition(ctx, pod, state); captureErr != nil {
			return captureErr
		}
	} else if state.node == nil || !state.node.Phase.Fulfilled(state.node.TaskResultSynced) || state.workflow.Status.IsTaskResultIncomplete(state.node.ID) {
		return StatusCaptureHold(CaptureResultPending, "pod cleanup is awaiting the persisted node result")
	}
	policy := state.workflow.GetExecSpec().PodGC
	selector, err := policy.GetLabelSelector()
	if err != nil {
		return fmt.Errorf("pod cleanup selector: %w", err)
	}
	delay := time.Duration(0)
	if c.config != nil {
		delay = c.config.GetPodGCDeleteDelayDuration()
	}
	if configured, err := policy.GetDeleteDelayDuration(); err != nil {
		return err
	} else if configured >= 0 {
		delay = configured
	}
	c.EnactAnyPodCleanup(ctx, selector, pod, policy.GetStrategy(), state.workflow.Status.Phase, delay)
	return nil
}
