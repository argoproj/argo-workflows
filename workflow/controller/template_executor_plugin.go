package controller

import (
	"reflect"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

func (woc *wfOperationCtx) runTemplateExecutorPlugins(tmpl *wfv1.Template, node *wfv1.NodeStatus) *wfv1.NodeStatus {
	req := plugins.ExecuteTemplateArgs{
		Workflow: woc.wf,
		Template: tmpl,
	}
	reply := &plugins.ExecuteTemplateReply{
		Node: node.DeepCopy(),
	}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.TemplateExecutor); ok {
			if err := plug.ExecuteTemplate(req, reply); err != nil {
				woc.markNodeError(node.Name, err)
			}
		}
	}
	if !reflect.DeepEqual(node, reply.Node) {
		woc.wf.Status.Nodes[node.ID] = *reply.Node
		woc.updated = true
	}
	return reply.Node
}
