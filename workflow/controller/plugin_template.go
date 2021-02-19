package controller

import (
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	pluginpkg "github.com/argoproj/argo-workflows/v3/pkg/plugin"
)

var templatePlugins = make(map[string]pluginpkg.TemplateExecutor) // templateType -> name

func (woc *wfOperationCtx) executePluginTemplate(nodeName string, orgTmpl wfv1.TemplateReferenceHolder, node *wfv1.NodeStatus, templateScope string, processedTmpl *wfv1.Template, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	if node != nil {
		return node, nil // don't run this twice
	}
	node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePlugin, templateScope, processedTmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	ty, _, err := processedTmpl.Plugin.Get()
	if err != nil {
		return nil, err
	}
	p, ok := templatePlugins[ty]
	if !ok {
		return nil, fmt.Errorf("no plugin for %q", ty)
	}
	woc.log.Infof("executing %q", ty)
	resp := &wfv1.NodeStatus{}
	err = p.ExecuteNode(
		pluginpkg.ExecuteNodeReq{
			Workflow: woc.wf.ObjectMeta,
			Template: *woc.wf.GetTemplateByName(node.TemplateName),
			Node:     *node,
		},
		resp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin template: %w", err)
	}
	if resp.Phase == "" {
		return node, nil
	}
	return woc.markNodePhase(nodeName, resp.Phase, resp.Message, resp.Outputs), nil
}
