package controller

import (
	"context"
	"encoding/json"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) addNodeToTaskSet(ctx context.Context, tmpl wfv1.Template, node wfv1.NodeStatus) error {
	obj, _, err := woc.controller.workflowTaskSetInformer.GetStore().GetByKey(woc.wf.Namespace + "/" + woc.wf.Name)
	if err != nil {
		return err
	}
	a, ok := obj.(*wfv1.WorkflowTaskSet)
	// con: scheduling complex
	// con: potentially resource too large for all data, would result is delay in work being done
	if ok {
		_, ok = a.Spec.Nodes[node.ID] // must (a) agent must exist and (b) agent must know about node
	}
	if ok {
		woc.log.Info("node already added to taskset")
		return nil
	}
	woc.log.WithField("nodeID", node.ID).Info("adding node to taskset")
	i := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace)
	x := &wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: woc.wf.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)), // make sure it is deleted when the workflow is deleted
			},
		},
		Spec: wfv1.WorkflowTaskSetSpec{
			Templates: []wfv1.Template{tmpl},
			Nodes:     wfv1.Nodes{node.ID: node},
		},
	}
	data, err := json.Marshal(x)
	if err != nil {
		return err
	}
	// con: consistency check
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
