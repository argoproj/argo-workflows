package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

func (woc *wfOperationCtx) executeDAG(ctx context.Context, nodeName string, tmplCtx *templateresolution.TemplateContext, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		_, node = woc.initializeExecutableNode(ctx, nodeName, wfv1.NodeTypeDAG, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning, opts.nodeFlag, true)
	}

	// If the node is already fulfilled (e.g. memoization cache hit), skip DAG execution
	if node.Fulfilled() {
		return node, nil
	}

	defer func() {
		deferNode, nodeErr := woc.wf.Status.Nodes.Get(node.ID)
		if nodeErr != nil {
			// CRITICAL ERROR IF THIS BRANCH IS REACHED -> PANIC
			panic(fmt.Sprintf("expected node for %s due to preceded initializeExecutableNode but couldn't find it", node.ID))
		}
		if deferNode.Fulfilled() {
			woc.killDaemonedChildren(ctx, deferNode.ID)
		}
	}()

	engine := NewEngine(woc, nodeName, tmplCtx, tmpl, orgTmpl, node.ID, opts.onExitTemplate)

	var tasks []dag.Task
	for i := range tmpl.DAG.Tasks {
		tasks = append(tasks, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[i]})
	}

	engine.Execute(ctx, tasks)
	return woc.wf.GetNodeByName(nodeName)
}
