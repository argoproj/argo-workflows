package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) patchTaskSet(ctx context.Context, patch interface{}, pathTypeType types.PatchType) error {
	patchByte, err := json.Marshal(patch)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	_, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Patch(ctx, woc.wf.Name, pathTypeType, patchByte, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed patching taskset: %v", err)
	}
	return nil
}

func (woc *wfOperationCtx) getDeleteTaskAndNodePatch() map[string]interface{} {
	deletedNode := make(map[string]interface{})
	for _, node := range woc.wf.Status.Nodes {
		if (node.Type == wfv1.NodeTypeHTTP || node.Type == wfv1.NodeTypePlugin) && node.Fulfilled() {
			deletedNode[node.ID] = nil
		}
	}

	// Delete the completed Tasks and nodes status
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"tasks": deletedNode,
		},
		"status": map[string]interface{}{
			"nodes": deletedNode,
		},
	}
	return patch
}
func taskSetNode(n wfv1.NodeStatus) bool {
	return n.Type == wfv1.NodeTypeHTTP || n.Type == wfv1.NodeTypePlugin
}

func (woc *wfOperationCtx) hasTaskSetNodes() bool {
	return woc.wf.Status.Nodes.Any(taskSetNode)
}

func (woc *wfOperationCtx) removeCompletedTaskSetStatus(ctx context.Context) error {
	if !woc.hasTaskSetNodes() {
		return nil
	}
	return woc.patchTaskSet(ctx, woc.getDeleteTaskAndNodePatch(), types.MergePatchType)
}

func (woc *wfOperationCtx) completeTaskSet(ctx context.Context) error {
	if !woc.hasTaskSetNodes() {
		return nil
	}
	patch := woc.getDeleteTaskAndNodePatch()
	patch["metadata"] = metav1.ObjectMeta{
		Labels: map[string]string{
			common.LabelKeyCompleted: "true",
		},
	}
	return woc.patchTaskSet(ctx, patch, types.MergePatchType)
}

func (woc *wfOperationCtx) getWorkflowTaskSet() (*wfv1.WorkflowTaskSet, error) {
	taskSet, exists, err := woc.controller.wfTaskSetInformer.Informer().GetIndexer().GetByKey(woc.wf.Namespace + "/" + woc.wf.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return taskSet.(*wfv1.WorkflowTaskSet), nil
}

func (woc *wfOperationCtx) taskSetReconciliation(ctx context.Context) {
	if err := woc.reconcileTaskSet(ctx); err != nil {
		woc.log.WithError(err).Error("error in workflowtaskset reconciliation")
		return
	}
	if err := woc.reconcileAgentPod(ctx); err != nil {
		woc.log.WithError(err).Error("error in agent pod reconciliation")
		woc.markWorkflowError(ctx, err)
		return
	}
}

func (woc *wfOperationCtx) nodeRequiresTaskSetReconciliation(nodeName string) bool {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		return false
	}
	// If this node is of type HTTP, it will need an HTTP reconciliation
	if taskSetNode(*node) {
		return true
	}
	for _, child := range node.Children {
		// If any of the node's children need an HTTP reconciliation, the parent node will also need one
		if woc.nodeRequiresTaskSetReconciliation(child) {
			return true
		}
	}
	// If neither of the children need one -- or if there are no children -- no HTTP reconciliation is needed.
	return false
}

func (woc *wfOperationCtx) reconcileTaskSet(ctx context.Context) error {
	workflowTaskSet, err := woc.getWorkflowTaskSet()
	if err != nil {
		return err
	}

	woc.log.Info("TaskSet Reconciliation")
	if workflowTaskSet != nil && len(workflowTaskSet.Status.Nodes) > 0 {
		for nodeID, taskResult := range workflowTaskSet.Status.Nodes {
			node := woc.wf.Status.Nodes[nodeID]

			node.Outputs = taskResult.Outputs.DeepCopy()
			node.Phase = taskResult.Phase
			node.Message = taskResult.Message
			node.FinishedAt = metav1.Now()

			woc.wf.Status.Nodes[nodeID] = node
			woc.updated = true
		}
	}
	return woc.createTaskSet(ctx)
}

func (woc *wfOperationCtx) createTaskSet(ctx context.Context) error {
	if len(woc.taskSet) == 0 {
		return nil
	}

	woc.log.Info("Creating TaskSet")
	taskSet := wfv1.WorkflowTaskSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflow.WorkflowTaskSetKind,
			APIVersion: workflow.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: woc.wf.Namespace,
			Name:      woc.wf.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: woc.wf.APIVersion,
					Kind:       woc.wf.Kind,
					UID:        woc.wf.UID,
					Name:       woc.wf.Name,
				},
			},
		},
		Spec: wfv1.WorkflowTaskSetSpec{
			Tasks: woc.taskSet,
		},
	}
	woc.log.Debug("creating new taskset")

	_, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Create(ctx, &taskSet, metav1.CreateOptions{})

	if apierr.IsConflict(err) || apierr.IsAlreadyExists(err) {
		woc.log.Debug("patching the exiting taskset")
		spec := map[string]interface{}{
			"metadata": metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyCompleted: strconv.FormatBool(woc.wf.Status.Fulfilled()),
				},
			},
			"spec": wfv1.WorkflowTaskSetSpec{Tasks: woc.taskSet},
		}
		// patch the new templates into taskset
		err = woc.patchTaskSet(ctx, spec, types.MergePatchType)
		if err != nil {
			woc.log.WithError(err).Error("Failed to patch WorkflowTaskSet")
			return fmt.Errorf("failed to patch TaskSet. %v", err)
		}
	} else if err != nil {
		woc.log.WithError(err).Error("Failed to create WorkflowTaskSet")
		return err
	}
	return nil
}
