package progress

import (
	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// UpdateProgress ensures the workflow's progress is updated with the individual node progress.
// This func can perform any repair work needed
func UpdateProgress(wf *wfv1.Workflow) {
	wf.Status.Progress = wfv1.ProgressZero
	// We loop over all executable nodes first, otherwise sum will be wrong.
	for nodeID, node := range wf.Status.Nodes {
		if !executable(node.Type) {
			continue
		}
		// all executable nodes should have progress defined, if not, we just set it to the default value.
		if node.Progress == wfv1.ProgressUndefined {
			node.Progress = wfv1.ProgressDefault
			wf.Status.Nodes.Set(nodeID, node)
		}
		// it could be possible for corruption to result in invalid progress, we just ignore invalid progress
		if !node.Progress.IsValid() {
			continue
		}
		// if the node has finished successfully, then we can just set progress complete
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped, wfv1.NodeOmitted:
			node.Progress = node.Progress.Complete()
			wf.Status.Nodes.Set(nodeID, node)
		}
		// the total should only contain node that are valid
		wf.Status.Progress = wf.Status.Progress.Add(node.Progress)
	}
	// For non-executable nodes, we sum up the children.
	// It is quite possible for a succeeded node to contain failed children (e.g. continues-on failed flag is set)
	// so it is possible for the sum progress to be "1/2" (for example)
	for nodeID, node := range wf.Status.Nodes {
		if executable(node.Type) {
			continue
		}
		progress := sumProgress(wf, node, make(map[string]bool))
		if progress.IsValid() {
			node.Progress = progress
			wf.Status.Nodes.Set(nodeID, node)
		}
	}
	// we could check an invariant here, wf.Status.Nodes[wf.Name].Progress == wf.Status.Progress, but I think there's
	// always the chance that the nodes get corrupted, so I think we leave it
}

// executable states that the progress of this node type is updated by other code. It should not be summed.
// It maybe that this type of node never gets progress.
func executable(nodeType wfv1.NodeType) bool {
	switch nodeType {
	case wfv1.NodeTypePod, wfv1.NodeTypeHTTP, wfv1.NodeTypePlugin, wfv1.NodeTypeContainer, wfv1.NodeTypeSuspend:
		return true
	default:
		return false
	}
}

func sumProgress(wf *wfv1.Workflow, node wfv1.NodeStatus, visited map[string]bool) wfv1.Progress {
	progress := wfv1.ProgressZero
	for _, childNodeID := range node.Children {
		if visited[childNodeID] {
			continue
		}
		visited[childNodeID] = true
		// this will tolerate missing child (will be "") and therefore ignored
		child, err := wf.Status.Nodes.Get(childNodeID)
		if err != nil {
			log.Warnf("Couldn't obtain child for %s, panicking", childNodeID)
			continue
		}
		progress = progress.Add(sumProgress(wf, *child, visited))
		if executable(child.Type) {
			v := child.Progress
			if v.IsValid() {
				progress = progress.Add(v)
			}
		}
	}
	return progress
}
