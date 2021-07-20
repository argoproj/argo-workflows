package controller

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) getAgentPodName() string {
	return woc.wf.NodeID("agent") + "-agent"
}

func (woc *wfOperationCtx) isAgentPod(pod *apiv1.Pod) bool {
	return pod.Name == woc.getAgentPodName()
}

func (woc *wfOperationCtx) reconcileAgentPod(ctx context.Context) error {
	woc.log.Infof("reconcileAgentPod")
	if len(woc.taskSet) == 0 {
		return nil
	}
	pod, err := woc.createAgentPod(ctx)
	if err != nil {
		return err
	}
	// Check Pod is just created
	if pod.Status.Phase != "" {
		woc.updateAgentPodStatus(pod)
	}
	return nil
}

func (woc *wfOperationCtx) updateAgentPodStatus(pod *apiv1.Pod) {
	woc.log.Infof("updateAgentPodStatus")
	for _, node := range woc.wf.Status.Nodes {
		// Update POD (Error and Failed) status to all HTTP Templates node status
		if node.Type == wfv1.NodeTypeHTTP {
			if newState := woc.assessAgentPodStatus(pod, &node); newState != nil {
				if woc.wf.Status.Nodes[node.ID].Phase != newState.Phase {
					woc.wf.Status.Nodes[node.ID] = *newState
					if newState.Fulfilled() {
						woc.addOutputsToGlobalScope(newState.Outputs)
					}
				}
				woc.updated = true
			}
		}
	}
}

func (woc *wfOperationCtx) assessAgentPodStatus(pod *apiv1.Pod, node *wfv1.NodeStatus) *wfv1.NodeStatus {
	woc.log.Infof("assessAgentPodStatus")
	var newPhase wfv1.NodePhase
	var message string
	updated := false
	switch pod.Status.Phase {
	case apiv1.PodPending:
		newPhase = wfv1.NodePending
		message = getPendingReason(pod)
	case apiv1.PodSucceeded:
		newPhase = wfv1.NodeSucceeded
	case apiv1.PodFailed:
		newPhase, message = woc.inferFailedReason(pod)
		woc.log.WithField("displayName", node.DisplayName).WithField("templateName", node.TemplateName).
			WithField("pod", pod.Name).Infof("Pod failed: %s", message)
	case apiv1.PodRunning:
		newPhase = wfv1.NodeRunning
	default:
		newPhase = wfv1.NodeError
		message = fmt.Sprintf("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
	}

	if !node.Fulfilled() && (node.Phase != newPhase) {
		woc.log.Infof("Updating node %s status %s -> %s", node.ID, node.Phase, newPhase)
		// if we are transitioning from Pending to a different state, clear out pending message
		if node.Phase == wfv1.NodePending {
			node.Message = ""
		}
		updated = true
		node.Phase = newPhase
	}

	if message != "" && node.Message != message {
		woc.log.Infof("Updating node %s message: %s", node.ID, message)
		updated = true
		node.Message = message
	}

	if node.Fulfilled() && node.FinishedAt.IsZero() {
		updated = true
		node.FinishedAt = getLatestFinishedAt(pod)
	}

	if updated {
		return node
	}
	return nil
}

func (woc *wfOperationCtx) createAgentPod(ctx context.Context) (*apiv1.Pod, error) {
	podName := woc.getAgentPodName()

	obj, exists, err := woc.controller.podInformer.GetStore().Get(cache.ExplicitKey(woc.wf.Namespace + "/" + podName))

	if err != nil {
		return nil, fmt.Errorf("failed to get pod from informer store: %w", err)
	}

	if exists {
		existing, ok := obj.(*apiv1.Pod)
		if ok {
			woc.log.WithField("podPhase", existing.Status.Phase).Debugf("Skipped pod %s  creation: already exists", podName)
			return existing, nil
		}
	}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: woc.wf.ObjectMeta.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.ObjectMeta.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",                // Allows filtering by incomplete workflow pods
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:    apiv1.RestartPolicyNever,
			ImagePullSecrets: woc.execWf.Spec.ImagePullSecrets,
			Containers: []apiv1.Container{
				{
					Name:            "main",
					Command:         []string{"argoexec"},
					Args:            []string{"agent"},
					Image:           woc.controller.executorImage(),
					ImagePullPolicy: apiv1.PullIfNotPresent,
					Env: []apiv1.EnvVar{
						{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
					},
				},
			},
		},
	}

	if woc.controller.Config.InstanceID != "" {
		pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}
	if woc.wf.Spec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = woc.wf.Spec.ServiceAccountName
	}

	woc.log.Debugf("Creating Agent Pod: %s", podName)

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			woc.log.Infof("Failed pod %s  creation: already exists", podName)
			return created, nil
		}
		if errorsutil.IsTransientErr(err) {
			return nil, err
		}
		woc.log.Infof("Failed to create Agent pod %s: %v", podName, err)
		return nil, errors.InternalWrapError(err)
	}
	woc.log.Infof("Created Agent pod: %s (%s)", podName, created.Name)
	return created, nil
}
