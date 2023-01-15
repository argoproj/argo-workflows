package controller

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/workflow/common"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

func (woc *wfOperationCtx) isOnWorkflowCompleteStrategy() bool {
	return woc.orig.Spec.PodGC != nil && woc.orig.Spec.PodGC.Strategy == wfv1.PodGCOnWorkflowCompletion
}

// @podGC is used to generate the label selector @labelPodCompleted is used to determine if
// we should add a common.LabelKeyCompleted=true label
func getLabelSelector(wfName string, podGC *wfv1.PodGC, mergePodGCLabels bool, labelCompleted *bool) metav1.LabelSelector {
	// upper limit on memory usage set
	labels := make(map[string]string)
	labels[common.LabelKeyWorkflow] = wfName
	if labelCompleted != nil {
		boolStr := "false"
		if *labelCompleted {
			boolStr = "true"
		}
		labels[common.LabelKeyCompleted] = boolStr
	}

	expressions := []metav1.LabelSelectorRequirement{}

	if mergePodGCLabels && podGC != nil && podGC.LabelSelector != nil {
		for key, value := range podGC.LabelSelector.MatchLabels {
			labels[key] = value
		}
		expressions = append(expressions, podGC.LabelSelector.MatchExpressions...)
	}
	return metav1.LabelSelector{MatchLabels: labels, MatchExpressions: expressions}
}

func (woc *wfOperationCtx) runOnWorkflowCompleteCleanup(ns string, podGC *wfv1.PodGC) error {
	if woc.finishedOnWorkflowCompleteCleanup {
		return nil
	}

	// canot run if workflow not fulfilled
	if !woc.wf.Status.Fulfilled() {
		return nil
	}

	markDone := true

	pods := woc.controller.kubeclientset.CoreV1().Pods(ns)
	wfName := woc.execWf.ObjectMeta.Name

	rateLimiter := &woc.rateLimiter

	rateLimiter.Wait()

	notCompletedValue := false
	lselector := getLabelSelector(wfName, podGC, false, &notCompletedValue)
	selector, err := metav1.LabelSelectorAsSelector(&lselector)
	if err != nil {
		log.Errorf("unable to create selector due to %s", err)
		return err
	}

	selectorString := selector.String()
	podList, err := pods.List(context.Background(), metav1.ListOptions{LabelSelector: selectorString})
	if err != nil {
		log.Errorf("unable to obtain list of pods due to %s", err)
		return err
	}

	for _, pod := range podList.Items {
		rateLimiter.Wait()
		_, err := pods.Patch(
			context.Background(),
			pod.Name,
			types.MergePatchType,
			[]byte(`{"metadata": {"labels": {"workflows.argoproj.io/completed": "true"}}}`),
			metav1.PatchOptions{},
		)
		if err != nil {
			log.Errorf("was not able to patch pods with completed as true for pods %s in workflow %s due to %s", pod.Name, woc.execWf.ObjectMeta.Name, err)
			markDone = false
		}
	}

	labelCompletedValue := true
	lselector = getLabelSelector(wfName, podGC, true, &labelCompletedValue)
	selector, err = metav1.LabelSelectorAsSelector(&lselector)
	if err != nil {
		log.Errorf("was not able to generate a selector due to %s hence abandoning deletion", err)
		return err
	}

	selectorString = selector.String()
	rateLimiter.Wait()
	err = pods.DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: selectorString})
	if err != nil {
		log.Errorf("was not able to delete collection to due %s", err)
		return err
	}

	if markDone {
		woc.finishedOnWorkflowCompleteCleanup = true
	}
	return nil
}

// clean up pods by either adding it to the cleanup queue or if
// the PodGC.Strategy is OnWorkflowCompletion then wait till the workflow is completed
func (woc *wfOperationCtx) cleanupPods() {
	woc.log.Infof("pod cleanup call issued")
	delay := woc.controller.Config.GetPodGCDeleteDelayDuration()
	podGC := woc.execWf.Spec.PodGC
	strategy := podGC.GetStrategy()
	selector, _ := podGC.GetLabelSelector()
	workflowPhase := woc.wf.Status.Phase
	log := woc.log.WithFields(logrus.Fields{"podGC": podGC, "strategy": strategy, "selector": selector.String(), "wfPhase": workflowPhase})
	objs, _ := woc.controller.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)

	// specific handling for OnWorkflowCompletion
	// we will be creating unnecessary DeleteCollection requests after the first one
	// to fix this, we would need a variable in wfOperationCtx. I think it is possibly okay to avoid this optimisation for now.
	if woc.isOnWorkflowCompleteStrategy() {
		err := woc.runOnWorkflowCompleteCleanup(woc.wf.Namespace, podGC)
		if err != nil {
			log.Error("abandoning cleanup due to error")
		}
		return
	}

	for _, obj := range objs {
		pod := obj.(*apiv1.Pod)
		if _, ok := pod.Labels[common.LabelKeyComponent]; ok { // for these types we don't want to do PodGC
			continue
		}
		nodeID := woc.nodeID(pod)
		if !woc.wf.Status.Nodes[nodeID].Phase.Fulfilled() {
			continue
		}

		log.Infof("cleaning up pod %s if applicable", pod.Name)
		log := log.WithField("podName", pod.Name)

		action := determinePodCleanupAction(selector, pod.Labels, strategy, workflowPhase, pod.Status.Phase)
		log.Infof("got pod cleanup action %s", action)
		switch action {
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
