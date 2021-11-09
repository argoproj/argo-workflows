package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runNodePreExecutePlugins(tmpl *wfv1.Template, node *wfv1.NodeStatus) error {
	println("ALEX", wfv1.MustMarshallJSON(node))
	args := plugins.NodePreExecuteArgs{Workflow: woc.wf.Reduced(), Template: tmpl, Node: node}
	reply := &plugins.NodePreExecuteReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.NodeLifecycleHook); ok {
			if err := plug.NodePreExecute(args, reply); err != nil {
				return err
			} else if reply.Node != nil {
				if err := woc.patchObj(node, reply.Node); err != nil {
					return err
				}
				woc.wf.Status.Nodes[node.ID] = *node
				woc.updated = true
			}
		}
	}
	return nil
}

func (woc *wfOperationCtx) runNodePostExecutePlugins(tmpl *wfv1.Template, old, new *wfv1.NodeStatus) error {
	args := plugins.NodePostExecuteArgs{Workflow: woc.wf.Reduced(), Template: tmpl, Old: old, New: new}
	reply := &plugins.NodePostExecuteReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.NodeLifecycleHook); ok {
			if err := plug.NodePostExecute(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}
