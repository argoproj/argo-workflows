package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// eachPooledNode calls f with each of the workflow's pooled nodes for which
// match is true, as the node is when f is called. Every node is matched
// before f is first called, so f may update nodes.
func (woc *wfOperationCtx) eachPooledNode(match func(*wfv1.NodeStatus) bool, f func(*wfv1.NodeStatus)) {
	var ids []string
	for id, node := range woc.wf.Status.Nodes {
		if isPooled(&node) && match(&node) {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if node, err := woc.wf.Status.Nodes.Get(id); err == nil {
			f(node)
		}
	}
}
