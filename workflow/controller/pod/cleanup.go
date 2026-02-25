package pod

import (
	"context"
	"slices"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v4/workflow/common"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func (c *Controller) EnactAnyPodCleanup(
	ctx context.Context,
	selector labels.Selector,
	pod *apiv1.Pod,
	strategy wfv1.PodGCStrategy,
	workflowPhase wfv1.WorkflowPhase,
	delay time.Duration,
) {
	action := determinePodCleanupAction(selector, pod.Labels, strategy, workflowPhase, pod.Status.Phase, pod.Finalizers)
	switch action {
	case noAction: // ignore
		break
	case deletePod:
		c.queuePodForCleanupAfter(ctx, pod.Namespace, pod.Name, action, delay)
	default:
		c.queuePodForCleanup(ctx, pod.Namespace, pod.Name, action)
	}

}

func determinePodCleanupAction(
	selector labels.Selector,
	podLabels map[string]string,
	strategy wfv1.PodGCStrategy,
	workflowPhase wfv1.WorkflowPhase,
	podPhase apiv1.PodPhase,
	finalizers []string,
) podCleanupAction {
	switch {
	case !selector.Matches(labels.Set(podLabels)): // if the pod will never be deleted, label it now
		return labelPodCompleted
	case strategy == wfv1.PodGCOnPodNone:
		return labelPodCompleted
	case strategy == wfv1.PodGCOnWorkflowCompletion && workflowPhase.Completed():
		return deletePod
	case strategy == wfv1.PodGCOnWorkflowSuccess && workflowPhase == wfv1.WorkflowSucceeded:
		return deletePod
	case strategy == wfv1.PodGCOnPodCompletion:
		return deletePod
	case strategy == wfv1.PodGCOnPodSuccess && podPhase == apiv1.PodSucceeded:
		return deletePod
	case strategy == wfv1.PodGCOnPodSuccess && podPhase == apiv1.PodFailed:
		return labelPodCompleted
	case workflowPhase.Completed():
		return labelPodCompleted
	case hasOurFinalizer(finalizers):
		return removeFinalizer
	}
	return noAction
}

func hasOurFinalizer(finalizers []string) bool {
	if finalizers != nil {
		return slices.Contains(finalizers, common.FinalizerPodStatus)
	}
	return false
}
