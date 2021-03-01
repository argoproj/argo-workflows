package controller

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) scheduleNodeOnAgent(ctx context.Context, tmpl wfv1.Template, node wfv1.NodeStatus) error {
	if err := woc.createAgent(ctx); err != nil {
		return err
	}
	if err := woc.scheduleNodeOnAgentUsingWorkflowAgent(ctx, tmpl, node); err != nil {
		return err
	}
	return woc.scheduleNodeOnAgentUsingWorkflowNode(ctx, tmpl, node)
}

func (woc *wfOperationCtx) scheduleNodeOnAgentUsingWorkflowAgent(ctx context.Context, tmpl wfv1.Template, node wfv1.NodeStatus) error {
	obj, _, err := woc.controller.workflowAgentInformer.GetStore().GetByKey(woc.wf.Namespace + "/" + woc.wf.Name)
	if err != nil {
		return err
	}
	a, ok := obj.(*wfv1.WorkflowAgent)
	// con: scheduling complex
	// con: potentially resource too large for all data, would result is delay in work being done
	if ok {
		_, ok = a.Spec.Nodes[node.ID] // must (a) agent must exist and (b) agent must know about node
	}
	if ok {
		woc.log.Info("node already scheduled an agent using workflow agent")
		return nil
	}
	woc.log.Info("scheduling node on agent using workflow agent")
	i := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowAgents(woc.wf.Namespace)
	x := &wfv1.WorkflowAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: woc.wf.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)), // make sure it is deleted when the workflow is deleted
			},
		},
		Spec: wfv1.WorkflowAgentSpec{
			Templates: []wfv1.Template{tmpl},
			Nodes:     wfv1.Nodes{node.ID: node},
		},
	}
	data, err := json.Marshal(x)
	if err != nil {
		return err
	}
	if _, err := i.Patch(ctx, woc.wf.Name, types.MergePatchType, data, metav1.PatchOptions{}); err != nil {
		if apierr.IsNotFound(err) {
			if _, err := i.Create(ctx, x, metav1.CreateOptions{}); err != nil { // already exist cannot happen here - patch would have been successful
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (woc *wfOperationCtx) scheduleNodeOnAgentUsingWorkflowNode(ctx context.Context, tmpl wfv1.Template, node wfv1.NodeStatus) error {
	_, exists, err := woc.controller.workflowNodeInformer.GetStore().GetByKey(woc.wf.Namespace + "/" + node.ID)
	if err != nil {
		return err
	}
	if exists {
		woc.log.Info("node already scheduled an agent using workflow node")
		return nil
	}
	woc.log.Info("scheduling node on agent using workflow node")
	i := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowNodes(woc.wf.Namespace)
	x := &wfv1.WorkflowNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:   node.ID,
			Labels: map[string]string{common.LabelKeyWorkflow: woc.wf.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)), // make sure it is deleted when the workflow is deleted
			},
		},
		Spec: &tmpl,
	}
	// con: N nodes per workflow
	if _, err := i.Create(ctx, x, metav1.CreateOptions{}); err != nil && !apierr.IsAlreadyExists(err) { // already exist can happen very rarely - if informer is behind
		return err
	}
	return nil
}

func (woc *wfOperationCtx) createAgent(ctx context.Context) error {
	podName := woc.agentPodName()

	_, exists, err := woc.controller.podInformer.GetStore().GetByKey(woc.wf.Namespace + "/" + podName)
	if err != nil {
		return err
	}
	if exists {
		woc.log.Info("agent already exists")
		return nil
	}

	tmpl := &wfv1.Template{} // dummy template
	c := woc.newExecContainer(common.MainContainerName, tmpl)
	c.Command = []string{"argoexec", "agent"}

	woc.log.Info("creating agent")

	_, err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				common.LabelKeyCompleted: "false", // should always be false, needed so it appear in the pod informer
			},
			Annotations: map[string]string{
				common.AnnotationKeyTemplate: `{}`, // blank entry to prevent panic
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)), // make sure it is deleted when the workflow is deleted
			},
		},
		Spec: corev1.PodSpec{
			ShareProcessNamespace: pointer.BoolPtr(woc.getContainerRuntimeExecutor() == common.ContainerRuntimeExecutorPNS),
			Volumes:               woc.createVolumes(tmpl),
			Containers:            []corev1.Container{*c},
			RestartPolicy:         corev1.RestartPolicyOnFailure, // the agent is allowed to exit with code 0 when it is done
		},
	}, metav1.CreateOptions{})

	if err != nil && !apierr.IsAlreadyExists(err) { // already exist can happen very rarely - if informer is behind
		return err
	}

	return nil
}

func (woc *wfOperationCtx) deleteAgent(ctx context.Context) error {
	namespace := woc.wf.Namespace
	name := woc.wf.Name
	logCtx := log.WithFields(log.Fields{"namespace": namespace, "name": name})
	podName := woc.agentPodName()
	_, exists, _ := woc.controller.podInformer.GetStore().GetByKey(namespace + "/" + podName)
	if !exists {
		logCtx.Info("no agent to delete")
		return nil
	}
	logCtx.Info("deleting agent")
	err := woc.controller.kubeclientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return err
	}
	return nil
}

func (woc *wfOperationCtx) agentPodName() string {
	return woc.wf.Name + "-agent"
}
