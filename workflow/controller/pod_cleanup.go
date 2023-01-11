package controller

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/workflow/common"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

func (woc *wfOperationCtx) isPodGCOnWorkflowCompletion() bool {
	return woc.orig.Spec.PodGC != nil && woc.orig.Spec.PodGC.Strategy == wfv1.PodGCOnWorkflowCompletion
}

// This will delete all failed/errored out pods on workflow completion but it will optimistically delete
// completed/successful pods. This avoids the work queues since large Workflows can actually cause the controller to crash
// due to OOM
func (woc *wfOperationCtx) runWorkflowCompletionCleanupStrategy(pod *apiv1.Pod, workflowPhase wfv1.WorkflowPhase) error {
	log := woc.log.WithField("podName", pod.Name)
	log.Infof("Running PodGC strategy PodGCOnWorkflowCompletion for %s", pod.Name)
	pods := woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace)

	// we can delete all the pods associated with this workflow now
	if woc.wf.Status.Fulfilled() {
		lselector := metav1.LabelSelector{MatchLabels: map[string]string{common.LabelKeyWorkflow: woc.execWf.ObjectMeta.Name}}
		selector, err := metav1.LabelSelectorAsSelector(&lselector)
		if err != nil {
			log.Errorf("was not able to generate a selector due to %s hence abandoning deletion", err)
			// probably best to do nothing in this case
			return err
		}
		selectorString := selector.String()
		log.Infof("workflow is fullfilled, deleting all pods now with selector %s", selectorString)
		err = pods.DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: selectorString})
		if err != nil {
			log.Errorf("was not able to delete collection to due %s", err)
		}
	} else {
		lselector := metav1.LabelSelector{MatchLabels: map[string]string{common.LabelKeyWorkflow: woc.execWf.ObjectMeta.Name, common.LabelKeyCompleted: "true"}}
		selector, err := metav1.LabelSelectorAsSelector(&lselector)
		if err != nil {
			log.Errorf("was not able to generate a selector due to %s hence abandoning deletion", err)
			// probably best to do nothing in this case
			return err
		}
		selectorString := selector.String()
		if determinePodCleanupAction(selector, pod.Labels, wfv1.PodGCOnWorkflowCompletion, workflowPhase, pod.Status.Phase) == labelPodCompleted {
			// we can still manually label pods as completed
			_, err := pods.Patch(
				context.Background(),
				pod.Name,
				types.MergePatchType,
				[]byte(`{"metadata": {"labels": {"workflows.argoproj.io/completed": "true"}}}`),
				metav1.PatchOptions{},
			)
			if err != nil {
				log.Errorf("was not able to patch pods with completed as true for pods %s in workflow %s due to %s hence abandoning deletion", pod.Name, woc.execWf.ObjectMeta.Name, err)
				return err
			}
		}
		// delete all the ones we have labelled as an optimisation
		err = pods.DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: selectorString})
		if err != nil {
			log.Errorf("was not able to delete collection due to %s", err)
		}
	}
	return nil
}

// clean up pods by either adding it to the cleanup queue or if
// the PodGC.Strategy is OnWorkflowCompletion then wait till the workflow is completed
func (woc *wfOperationCtx) cleanupPods() {
	delay := woc.controller.Config.GetPodGCDeleteDelayDuration()
	podGC := woc.execWf.Spec.PodGC
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
		if !woc.wf.Status.Nodes[nodeID].Phase.Fulfilled() {
			continue
		}

		// specific handling for OnWorkflowCompletion
		if woc.isPodGCOnWorkflowCompletion() {
			woc.runWorkflowCompletionCleanupStrategy(pod, workflowPhase)
			return
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
