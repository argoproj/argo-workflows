package controller

import (
	"context"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
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

	engine := NewEngine(woc, nodeName, tmplCtx, tmpl, node.ID, opts.onExitTemplate)

	var tasks []dag.Task
	known := make(map[string]bool, len(tmpl.DAG.Tasks))
	for i := range tmpl.DAG.Tasks {
		tasks = append(tasks, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[i]})
		known[tmpl.DAG.Tasks[i].Name] = true
	}

	// dag.target may be parameterised (validate.go's validateDAGTargets
	// skips it), so a target naming no task can only be caught here, once
	// substituted. dag.PullOrder silently drops a target it can't find, so
	// left unchecked the boundary would never see a node for it and
	// assessDAGPhase (target with no node) would keep it Running forever;
	// main panicked here (recovered by the operator into an Error), so this
	// keeps main's Error outcome without the panic.
	for name := range strings.FieldsSeq(tmpl.DAG.Target) {
		if !known[name] {
			woc.markNodeError(ctx, nodeName, fmt.Errorf("target '%s' is not defined", name))
			return woc.wf.GetNodeByName(nodeName)
		}
	}

	engine.Execute(ctx, dag.PullOrder(tasks, strings.Fields(tmpl.DAG.Target)))
	return woc.wf.GetNodeByName(nodeName)
}
