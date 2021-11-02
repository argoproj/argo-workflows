package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

func (woc *wfOperationCtx) runNodePreExecutePlugins(tmpl *wfv1.Template, node *wfv1.NodeStatus) *wfv1.NodeStatus {
	args := plugins.NodePreExecuteArgs{Workflow: woc.wf, Template: tmpl, Node: node}
	reply := &plugins.NodePreExecuteReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.NodeLifecycleHook); ok {
			if err := plug.NodePreExecute(args, reply); err != nil {
				woc.markNodeError(node.Name, err)
			} else if reply.Node != nil {
				woc.wf.Status.Nodes[reply.Node.ID] = *reply.Node
				woc.updated = true
			}
		}
	}
	return reply.Node
}

func (woc *wfOperationCtx) runNodePostExecutePlugins(tmpl *wfv1.Template, node *wfv1.NodeStatus) (*wfv1.NodeStatus, error) {
	args := plugins.NodePostExecuteArgs{Workflow: woc.wf, Template: tmpl, Node: node}
	reply := &plugins.NodePostExecuteReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.NodeLifecycleHook); ok {
			if err := plug.NodePostExecute(args, reply); err != nil {
				return node, err
			}
		}
	}
	return node, nil
}
