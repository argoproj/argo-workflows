package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
)

func (woc *wfOperationCtx) mergePatchTaskSet(ctx context.Context, patch any, subresources ...string) error {
	patchByte, err := json.Marshal(patch)
	if err != nil {
		return argoerrors.InternalWrapError(err)
	}
	_, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Patch(ctx, woc.wf.Name, types.MergePatchType, patchByte, metav1.PatchOptions{}, subresources...)
	if err != nil {
		return fmt.Errorf("failed patching taskset: %w", err)
	}
	return nil
}

func (woc *wfOperationCtx) getDeleteTaskAndNodePatch() (tasksPatch map[string]any, nodesPatch map[string]any) {
	deletedNode := make(map[string]any)
	for _, node := range woc.wf.Status.Nodes {
		if node.IsTaskSetNode() && node.Fulfilled() {
			deletedNode[node.ID] = nil
		}
	}

	// Delete the completed Tasks and nodes status
	tasksPatch = map[string]any{
		"spec": map[string]any{
			"tasks": deletedNode,
		},
	}
	nodesPatch = map[string]any{
		"status": map[string]any{
			"nodes": deletedNode,
		},
	}
	return
}

func (woc *wfOperationCtx) markTaskSetNodesError(ctx context.Context, err error) {
	for _, node := range woc.wf.Status.Nodes {
		if node.IsTaskSetNode() && !node.Fulfilled() {
			woc.markNodeError(ctx, node.Name, err)
		}
	}
}

func (woc *wfOperationCtx) hasTaskSetNodes() bool {
	return woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.IsTaskSetNode()
	})
}

func (woc *wfOperationCtx) removeCompletedTaskSetStatus(ctx context.Context) error {
	if !woc.hasTaskSetNodes() {
		return nil
	}
	tasksPatch, nodesPatch := woc.getDeleteTaskAndNodePatch()
	if woc.wf.Status.Fulfilled() {
		tasksPatch["metadata"] = metav1.ObjectMeta{
			Labels: map[string]string{
				common.LabelKeyCompleted: "true",
			},
		}
	}
	if err := woc.mergePatchTaskSet(ctx, nodesPatch, "status"); err != nil {
		return err
	}
	return woc.mergePatchTaskSet(ctx, tasksPatch)
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
		woc.log.WithError(err).Error(ctx, "error in workflowtaskset reconciliation")
		return
	}
	if err := woc.reconcileAgentPod(ctx); err != nil {
		woc.log.WithError(err).Error(ctx, "error in agent pod reconciliation")
		woc.markTaskSetNodesError(ctx, fmt.Errorf(`create agent pod failed with reason:"%w"`, err))
		return
	}
}

func (woc *wfOperationCtx) nodeRequiresTaskSetReconciliation(ctx context.Context, nodeName string) bool {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		return false
	}
	// If this node is of type HTTP, it will need an HTTP reconciliation
	if node.IsTaskSetNode() {
		return true
	}
	for _, child := range node.Children {
		// If any of the node's children need an HTTP reconciliation, the parent node will also need one
		childNodeName, err := woc.wf.Status.Nodes.GetName(child)
		if err != nil {
			woc.log.WithField("nodeID", child).WithFatal().Error(ctx, "was unable to get child node name for nodeID")
			panic("unable to obtain child node name")
		}
		if woc.nodeRequiresTaskSetReconciliation(ctx, childNodeName) {
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

	woc.log.Info(ctx, "TaskSet Reconciliation")
	if workflowTaskSet != nil && len(workflowTaskSet.Status.Nodes) > 0 {
		for nodeID, taskResult := range workflowTaskSet.Status.Nodes {
			node, err := woc.wf.Status.Nodes.Get(nodeID)
			if err != nil {
				woc.log.Warn(ctx, "returning but assumed validity before")
				woc.log.WithField("nodeID", nodeID).Error(ctx, "was unable to obtain node for nodeID")
				return err
			}

			node.Outputs = taskResult.Outputs.DeepCopy()
			node.Phase = taskResult.Phase
			node.Message = taskResult.Message
			node.FinishedAt = metav1.Now()

			woc.wf.Status.Nodes.Set(ctx, nodeID, *node)
			if node.MemoizationStatus != nil && node.Succeeded() {
				c := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, node.MemoizationStatus.CacheName)
				err := c.Save(ctx, node.MemoizationStatus.Key, node.ID, node.Outputs)
				if err != nil {
					woc.log.WithFields(logging.Fields{"nodeID": node.ID}).WithError(err).Error(ctx, "Failed to save node outputs to cache")
				}
			}
			woc.updated = true
		}
	}
	return woc.createTaskSet(ctx)
}

func (woc *wfOperationCtx) createTaskSet(ctx context.Context) error {
	if len(woc.taskSet) == 0 {
		return nil
	}

	woc.log.Info(ctx, "Creating TaskSet")
	labels := map[string]string{}
	if woc.controller.Config.InstanceID != "" {
		labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}
	taskSet := wfv1.WorkflowTaskSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflow.WorkflowTaskSetKind,
			APIVersion: workflow.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: woc.wf.Namespace,
			Name:      woc.wf.Name,
			Labels:    labels,
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
	woc.log.Debug(ctx, "creating new taskset")

	_, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Create(ctx, &taskSet, metav1.CreateOptions{})

	if apierr.IsConflict(err) || apierr.IsAlreadyExists(err) {
		woc.log.Debug(ctx, "patching the exiting taskset")
		spec := map[string]any{
			"metadata": metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyCompleted: strconv.FormatBool(woc.wf.Status.Fulfilled()),
				},
			},
			"spec": wfv1.WorkflowTaskSetSpec{Tasks: woc.taskSet},
		}
		// patch the new templates into taskset
		err = woc.mergePatchTaskSet(ctx, spec)
		if err != nil {
			woc.log.WithError(err).Error(ctx, "Failed to patch WorkflowTaskSet")
			return fmt.Errorf("failed to patch TaskSet. %w", err)
		}
	} else if err != nil {
		woc.log.WithError(err).Error(ctx, "Failed to create WorkflowTaskSet")
		return err
	}
	return nil
}
