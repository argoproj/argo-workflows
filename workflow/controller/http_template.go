package controller

import (
	"context"
	"encoding/json"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) executeHTTP(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeHTTP, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}

	if node.Fulfilled() {
		return node, nil
	}

	{ // create/update thing
		obj, _, err := woc.controller.workflowThingInformer.GetStore().GetByKey(woc.wf.Namespace + "/" + woc.wf.Name)
		if err != nil {
			return nil, err
		}
		t, ok := obj.(*wfv1.WorkflowThing)
		if ok {
			_, ok = t.Status.Nodes[node.ID]
		}
		if !ok {
			i := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowThings(woc.wf.Namespace)
			x := &wfv1.WorkflowThing{
				ObjectMeta: metav1.ObjectMeta{Name: woc.wf.Name},
				Status:     wfv1.WorkflowThingStatus{Nodes: wfv1.Nodes{node.ID: *node}},
			}
			data, err := json.Marshal(x)
			if err != nil {
				return nil, err
			}
			if _, err := i.Patch(ctx, woc.wf.Name, types.MergePatchType, data, metav1.PatchOptions{}); err != nil {
				if apierr.IsNotFound(err) {
					if _, err := i.Create(ctx, x, metav1.CreateOptions{}); err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
		}
	}
	{ // create agent pod
		podName := woc.wf.Name + "-agent"
		_, exists, err := woc.controller.podInformer.GetStore().GetByKey(woc.wf.Namespace + "/" + podName)
		if err != nil {
			return nil, err
		}
		if !exists {
			c := woc.newExecContainer(common.MainContainerName, tmpl)
			c.Command = []string{"argoexec", "agent"}
			_, err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Create(ctx, &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podName,
					Labels: map[string]string{
						common.LabelKeyCompleted: "false",
						// TODO instance ID
					},
					Annotations: map[string]string{
						common.AnnotationKeyTemplate: `{}`, // blank entry to prevent panic
					},
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
					},
				},
				Spec: apiv1.PodSpec{
					ShareProcessNamespace: pointer.BoolPtr(woc.getContainerRuntimeExecutor() == common.ContainerRuntimeExecutorPNS),
					Volumes:               woc.createVolumes(tmpl),
					Containers:            []apiv1.Container{*c},
					RestartPolicy:         apiv1.RestartPolicyOnFailure, // if it is successful, that is fine
				},
			}, metav1.CreateOptions{})
			if err != nil && !apierr.IsAlreadyExists(err) {
				return woc.requeueIfTransientErr(err, node.Name)
			}
		}
	}
	return node, nil
}
