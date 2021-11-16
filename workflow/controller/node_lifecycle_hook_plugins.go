package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	controllerplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
	"github.com/argoproj/argo-workflows/v3/util/patch"
)

func (woc *wfOperationCtx) runNodePreExecutePlugins(tmpl *wfv1.Template, node *wfv1.NodeStatus) error {
	plugs := woc.controller.getControllerPlugins()
	args := controllerplugins.NodePreExecuteArgs{Workflow: &controllerplugins.Workflow{ObjectMeta: woc.wf.ObjectMeta}, Template: tmpl, Node: node}
	reply := &controllerplugins.NodePreExecuteReply{}
	for _, sym := range plugs {
		if plug, ok := sym.(controllerplugins.NodeLifecycleHook); ok {
			if err := plug.NodePreExecute(args, reply); err != nil {
				return err
			} else if reply.Node != nil {
				if err := patch.Obj(node, reply.Node); err != nil {
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
	plugs := woc.controller.getControllerPlugins()
	args := controllerplugins.NodePostExecuteArgs{Workflow: &controllerplugins.Workflow{ObjectMeta: woc.wf.ObjectMeta}, Template: tmpl, Old: old, New: new}
	reply := &controllerplugins.NodePostExecuteReply{}
	for _, plug := range plugs {
		if plug, ok := plug.(controllerplugins.NodeLifecycleHook); ok {
			if err := plug.NodePostExecute(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}
