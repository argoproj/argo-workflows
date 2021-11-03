package progress

import (
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// PodProgress reads the progress annotation of a pod and ensures it's valid and synced
// with the node status.
func PodProgress(pod *apiv1.Pod, node *wfv1.NodeStatus) wfv1.Progress {
	progress := wfv1.Progress("0/1")
	if node.Progress.IsValid() {
		progress = node.Progress
	}

	if annotation, ok := pod.Annotations[common.AnnotationKeyProgress]; ok {
		v, ok := wfv1.ParseProgress(annotation)
		if ok {
			progress = v
		}
	}
	if node.Fulfilled() {
		progress = progress.Complete()
	}
	return progress
}

// UpdateProgress ensures the workflow's progress is updated with the individual node progress.
func UpdateProgress(wf *wfv1.Workflow) {
	wf.Status.Progress = "0/0"
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod {
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
		if child.Type == wfv1.NodeTypePod {
			v := child.Progress
			if v.IsValid() {
				progress = progress.Add(v)
			}
		}
	}
	return progress
}
