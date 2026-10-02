package controller

import (
	"context"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

// Cleanup intents must outlive informer lag and Workflow completion filtering.
// Read the API object and its referenced node version rather than cache state.
func (wfc *WorkflowController) lookupWorkflowForPodCleanup(ctx context.Context, namespace, name string, hydrateNodes bool) (*wfv1.Workflow, error) {
	wf, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if hydrateNodes {
		if err := wfc.hydrator.Hydrate(ctx, wf); err != nil {
			return nil, err
		}
	}
	return wf, nil
}

// Resolve the hydrator at invocation time: configuration reload can replace
// its repository after the Pod controller callback has been installed.
func (wfc *WorkflowController) hydrateWorkflowForPodCleanup(ctx context.Context, wf *wfv1.Workflow) error {
	return wfc.hydrator.Hydrate(ctx, wf)
}

func (woc *wfOperationCtx) getPodGCDelay(ctx context.Context, podGC *wfv1.PodGC) time.Duration {
	delay := woc.controller.Config.GetPodGCDeleteDelayDuration()
	podGCDelay, err := podGC.GetDeleteDelayDuration()
	if err != nil {
		woc.log.WithError(err).Warn(ctx, "failed to parse podGC.deleteDelayDuration")
	} else if podGCDelay >= 0 {
		delay = podGCDelay
	}
	return delay
}

func (woc *wfOperationCtx) queuePodsForCleanup(ctx context.Context) {
	podGC := woc.execWf.Spec.PodGC
	delay := woc.getPodGCDelay(ctx, podGC)
	strategy := podGC.GetStrategy()
	selector, _ := podGC.GetLabelSelector()
	workflowPhase := woc.wf.Status.Phase
	objs, _ := woc.controller.PodController.GetPodsByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
	for _, obj := range objs {
		pod := obj.(*apiv1.Pod)
		owner := metav1.GetControllerOf(pod)
		if pod.Namespace != woc.wf.Namespace || owner == nil || owner.Kind != workflow.WorkflowKind || owner.APIVersion != workflow.APIVersion || owner.Name != woc.wf.Name || owner.UID != woc.wf.UID || pod.Labels[common.LabelKeyControllerInstanceID] != woc.wf.Labels[common.LabelKeyControllerInstanceID] {
			continue
		}

		if _, ok := pod.Labels[common.LabelKeyComponent]; ok { // for these types we don't want to do PodGC
			continue
		}
		nodeID := woc.nodeID(pod)
		node, err := woc.wf.Status.Nodes.Get(nodeID)
		if err != nil {
			woc.log.WithField("nodeID", nodeID).Error(ctx, "was unable to obtain node for nodeID")
			continue
		}
		if node.Phase == wfv1.NodePending && pod.Status.Phase == apiv1.PodFailed && node.RestartingPodUID == string(pod.UID) {
			woc.controller.PodController.DeletePodByUID(ctx, pod.Namespace, pod.Name, string(pod.UID))
			continue
		}
		nodePhase := node.Phase
		if !nodePhase.Fulfilled(node.TaskResultSynced) {
			continue
		}
		woc.controller.PodController.EnactAnyPodCleanup(ctx, selector, pod, strategy, workflowPhase, delay)
	}
}
