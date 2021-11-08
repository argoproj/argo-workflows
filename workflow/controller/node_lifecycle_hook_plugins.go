package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	controller2 "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runNodePreExecutePlugins(tmpl *wfv1.Template, node *wfv1.NodeStatus) error {
	args := controller2.NodePreExecuteArgs{Workflow: woc.wf.Reduced(), Template: tmpl, Node: node}
	reply := &controller2.NodePreExecuteReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller2.NodeLifecycleHook); ok {
			if err := plug.NodePreExecute(args, reply); err != nil {
				return err
			} else if reply.Node != nil {
				reply.Node.DeepCopyInto(node)
				woc.wf.Status.Nodes[reply.Node.ID] = *node
				woc.updated = true
			}
		}
	}
	return nil
}

func (woc *wfOperationCtx) runNodePostExecutePlugins(tmpl *wfv1.Template, old, new *wfv1.NodeStatus) error {
	args := controller2.NodePostExecuteArgs{Workflow: woc.wf.Reduced(), Template: tmpl, Old: old, New: new}
	reply := &controller2.NodePostExecuteReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller2.NodeLifecycleHook); ok {
			if err := plug.NodePostExecute(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}
