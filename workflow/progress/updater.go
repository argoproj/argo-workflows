package progress

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func UpdateProgress(wf *wfv1.Workflow) {
	wf.Status.Progress = "0/0"
	for nodeID, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod {
			continue
		}
		progress := wfv1.Progress("0/1")
		if node.Fulfilled() {
			progress = "1/1"
		}
		node.Progress = progress
		wf.Status.Nodes[nodeID] = node
		wf.Status.Progress = wf.Status.Progress.Add(progress)
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
		if child.Type == wfv1.NodeTypePod {
			v := child.Progress
			if v.IsValid() {
				progress = progress.Add(v)
			}
		}
	}
	return progress
}
