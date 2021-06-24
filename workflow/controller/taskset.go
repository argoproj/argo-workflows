package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) getAgentPodName() string {
	return woc.wf.NodeID("agent") + "-agent"
}

func (woc *wfOperationCtx) executeTaskSet(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeHTTP, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}
	mainCtr := woc.newExecContainer(common.MainContainerName, tmpl)
	mainCtr.Command = []string{"argoexec", "agent"}
	// Append Agent in end of pod name to differentiate from normal podname
	podName := woc.getAgentPodName()
	_, err := woc.createAgentPod(ctx, podName, []apiv1.Container{*mainCtr}, tmpl)
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}
	if node.Phase == wfv1.NodePending {
		err := woc.controller.taskSetManager.CreateTaskSet(ctx, woc.wf, node.ID, *tmpl)
		if err != nil {
			return nil, err
		}
	}
	return node, nil
}

func (woc *wfOperationCtx) taskSetReconciliation() error {
	taskSet, err := woc.getWorkflowTaskSet()
	if err != nil {
		return err
	}
	if taskSet == nil || len(taskSet.Status.Nodes) == 0 {
		return nil
	}
	for nodeID, taskResult := range taskSet.Status.Nodes {
		node, ok := woc.wf.Status.Nodes[nodeID]
		if !ok {
			continue
		}
		node.Outputs = taskResult.Outputs.DeepCopy()
		node.Phase = taskResult.Phase
		node.Message = taskResult.Message
		woc.wf.Status.Nodes[nodeID] = node
	}
	return nil
}

func isAgentPod(pod *apiv1.Pod) bool {
	return strings.HasSuffix(pod.Name, "-agent")
}

func (woc *wfOperationCtx) reconcileAgentNode(pod *apiv1.Pod) {
	for _, node := range woc.wf.Status.Nodes {
		// Update POD (Error and Failed) status to all HTTP Templates node status
		if node.Type == wfv1.NodeTypeHTTP {
			if newState := woc.assessNodeStatus(pod, &node); newState != nil {
				if newState.Fulfilled() {
					woc.wf.Status.Nodes[node.ID] = *newState
					woc.addOutputsToGlobalScope(newState.Outputs)
					woc.updated = true
				}
			}
		}
	}
}

func (woc *wfOperationCtx) createAgentPod(ctx context.Context, nodeName string, mainCtrs []apiv1.Container, tmpl *wfv1.Template) (*apiv1.Pod, error) {
	podName := woc.getAgentPodName()

	obj, exists, err := woc.controller.podInformer.GetStore().Get(cache.ExplicitKey(woc.wf.Namespace + "/" + podName))

	if err != nil {
		return nil, fmt.Errorf("failed to get pod from informer store: %w", err)
	}

	if exists {
		existing, ok := obj.(*apiv1.Pod)
		if ok {
			woc.log.WithField("podPhase", existing.Status.Phase).Debugf("Skipped pod %s (%s) creation: already exists", tmpl.Name, podName)
			return existing, nil
		}
	}

	for i, c := range mainCtrs {
		if c.Name == "" {
			c.Name = common.MainContainerName
		}
		// Allow customization of main container resources.
		if isResourcesSpecified(woc.controller.Config.MainContainer) {
			c.Resources = *woc.controller.Config.MainContainer.Resources.DeepCopy()
		}
		mainCtrs[i] = c
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
			Volumes:          woc.createVolumes(tmpl),
			ImagePullSecrets: woc.execWf.Spec.ImagePullSecrets,
		},
	}

	if woc.controller.Config.InstanceID != "" {
		pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}
	if woc.getContainerRuntimeExecutor() == common.ContainerRuntimeExecutorPNS {
		pod.Spec.ShareProcessNamespace = pointer.BoolPtr(true)
	}

	err = woc.setupServiceAccount(ctx, pod, tmpl)
	if err != nil {
		return nil, err
	}
	waitCtr := woc.newWaitContainer(tmpl)
	pod.Spec.Containers = append(pod.Spec.Containers, *waitCtr)

	pod.Spec.Containers = append(pod.Spec.Containers, mainCtrs...)

	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarTemplate, Value: wfv1.MustMarshallJSON(tmpl)},
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
		{Name: common.EnvVarPodName, Value: podName},
		{Name: common.EnvVarDeadline, Value: woc.getDeadline(&createWorkflowPodOpts{}).Format(time.RFC3339)},
	}

	for i, c := range pod.Spec.Containers {
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarContainerName, Value: c.Name})
		c.Env = append(c.Env, envVars...)
		pod.Spec.Containers[i] = c
	}

	// Set the container template JSON in pod annotations, which executor examines for things like
	// artifact location/path.
	pod.ObjectMeta.Annotations = map[string]string{}

	woc.log.Debugf("Creating Pod: %s (%s)", nodeName, podName)

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if the
			// controller fails to persist the workflow after creating the pod.
			woc.log.Infof("Failed pod %s (%s) creation: already exists", nodeName, podName)
			return created, nil
		}
		if errorsutil.IsTransientErr(err) {
			return nil, err
		}
		woc.log.Infof("Failed to create pod %s (%s): %v", nodeName, podName, err)
		return nil, errors.InternalWrapError(err)
	}
	woc.log.Infof("Created pod: %s (%s)", nodeName, created.Name)
	woc.activePods++
	return created, nil
}
