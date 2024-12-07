package controller

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v3/workflow/common"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

func (woc *wfOperationCtx) queuePodsForCleanup() {
	delay := woc.controller.Config.GetPodGCDeleteDelayDuration()
	podGC := woc.execWf.Spec.PodGC
	podGCDelay, err := podGC.GetDeleteDelayDuration()
	if err != nil {
		woc.log.WithError(err).Warn("failed to parse podGC.deleteDelayDuration")
	} else if podGCDelay >= 0 {
		delay = podGCDelay
	}
	strategy := podGC.GetStrategy()
	selector, _ := podGC.GetLabelSelector()
	workflowPhase := woc.wf.Status.Phase
	objs, _ := woc.controller.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
	for _, obj := range objs {
		pod := obj.(*apiv1.Pod)
		if _, ok := pod.Labels[common.LabelKeyComponent]; ok { // for these types we don't want to do PodGC
			continue
		}
		nodeID := woc.nodeID(pod)
		nodePhase, err := woc.wf.Status.Nodes.GetPhase(nodeID)
		if err != nil {
			woc.log.Errorf("was unable to obtain node for %s", nodeID)
			continue
		}
		if !nodePhase.Fulfilled() {
			continue
		}
		switch determinePodCleanupAction(selector, pod.Labels, strategy, workflowPhase, pod.Status.Phase) {
		case deletePod:
			woc.controller.queuePodForCleanupAfter(pod.Namespace, pod.Name, deletePod, delay)
		case labelPodCompleted:
			woc.controller.queuePodForCleanup(pod.Namespace, pod.Name, labelPodCompleted)
		}
	}
}

func determinePodCleanupAction(
	selector labels.Selector,
	podLabels map[string]string,
	strategy wfv1.PodGCStrategy,
	workflowPhase wfv1.WorkflowPhase,
	podPhase apiv1.PodPhase,
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
	}
	return ""
}
