package controller

import (
	"fmt"
	"reflect"

	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	pluginpkg "github.com/argoproj/argo/v3/pkg/plugin"
	"github.com/argoproj/argo/v3/plugins"
)

var templatePlugins = make(map[string]string) // type -> name

func (woc *wfOperationCtx) reconcilePluginTemplates() {
	for _, node := range woc.wf.Status.Nodes.Filter(func(node wfv1.NodeStatus) bool { return node.Type == wfv1.NodeTypePlugin && !node.Phase.Fulfilled() }) {
		tmpl := *woc.wf.GetTemplateByName(node.TemplateName)
		ty, err := tmpl.Plugin.GetType()
		if err != nil {
			woc.markNodeError(node.Name, err)
			continue
		}
		name, ok := templatePlugins[ty]
		if !ok {
			woc.markNodeError(node.Name, fmt.Errorf("no plugin for %q", ty))
			continue
		}
		p := plugins.Plugins[name]
		symbol, err := p.Lookup("ReconcileNode")
		if err != nil {
			woc.log.WithError(err).Infof("plugin %q does not have symbol ReconcileNode", name)
			continue
		}
		f, ok := symbol.(pluginpkg.ReconcileNodeFunc)
		if !ok {
			woc.markNodeError(node.Name, fmt.Errorf("plugin %q symbol ReconcileNode is not ReconcileNodeFunc: %v", name, reflect.TypeOf(symbol).String()))
			continue
		}
		woc.log.Infof("reconcilling %q", name)
		resp := &wfv1.NodeStatus{}
		err = f(
			pluginpkg.ReconcileNodeReq{
				Workflow: woc.wf.ObjectMeta,
				Template: tmpl,
				Node:     node,
			},
			resp,
		)
		if err != nil {
			woc.markNodeError(node.Name, err)
		} else if node.Phase != "" {
			woc.markNodePhase(node.Name, resp.Phase, resp.Message, resp.Outputs)
		}
	}
}

func (woc *wfOperationCtx) executePluginTemplate(nodeName string, orgTmpl wfv1.TemplateReferenceHolder, node *wfv1.NodeStatus, templateScope string, processedTmpl *wfv1.Template, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	if node == nil {
		return node, nil // don't run this twice
	}
	node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePlugin, templateScope, processedTmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	ty, err := processedTmpl.Plugin.GetType()
	if err != nil {
		return nil, err
	}
	name, ok := templatePlugins[ty]
	if !ok {
		return nil, fmt.Errorf("no plugin for %q", ty)
	}
	p := plugins.Plugins[name]
	symbol, err := p.Lookup("ExecuteNode")
	if err != nil {
		return nil, fmt.Errorf("plugin %q does not have symbol ExecuteNode", name)
	}
	f, ok := symbol.(pluginpkg.ExecuteNodeFunc)
	if !ok {
		return woc.markNodeError(node.Name, fmt.Errorf("plugin %q symbol ExecuteNode is not ExecuteNodeFunc: %v", name, reflect.TypeOf(symbol).String())), nil
	}
	woc.log.Infof("executing %q", name)
	resp := &wfv1.NodeStatus{}
	err = f(
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
