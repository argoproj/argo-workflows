package progress

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// UpdateProgress ensures the workflow's progress is updated with the individual node progress.
func UpdateProgress(wf *wfv1.Workflow) {
	wf.Status.Progress = "0/0"
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod && node.Type != wfv1.NodeTypeHTTP {
			continue
		}
		if node.Progress.IsValid() {
			wf.Status.Progress = wf.Status.Progress.Add(node.Progress)
		}
	}
	for nodeID, node := range wf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod {
			continue
		}
		progress := sumProgress(wf, node, make(map[string]bool))
		if progress.IsValid() && node.Progress != progress {
			node.Progress = progress
			wf.Status.Nodes[nodeID] = node
		}
	}
}

func sumProgress(wf *wfv1.Workflow, node wfv1.NodeStatus, visited map[string]bool) wfv1.Progress {
	progress := wfv1.Progress("0/0")
	for _, childNodeID := range node.Children {
		if visited[childNodeID] {
			continue
		}
		visited[childNodeID] = true
		// this will tolerate missing child (will be "") and therefore ignored
		child := wf.Status.Nodes[childNodeID]
		progress = progress.Add(sumProgress(wf, child, visited))
		if child.Type == wfv1.NodeTypePod || child.Type == wfv1.NodeTypeHTTP {
			v := child.Progress
			if v.IsValid() {
				progress = progress.Add(v)
			}
		}
	}
	return progress
}
