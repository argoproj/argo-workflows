package resource

import (
	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func UpdateResourceDurations(wf *wfv1.Workflow) {
	wf.Status.ResourcesDuration = wfv1.ResourcesDuration{}
	for nodeID, node := range wf.Status.Nodes {
		// pods are already calculated and so we do not need to compute them,
		// AND they are the only node that should contribute to the total
		if node.Type == wfv1.NodeTypePod {
			wf.Status.ResourcesDuration = wf.Status.ResourcesDuration.Add(node.ResourcesDuration)
		} else if node.Fulfilled() {
			// compute the sum of all children
			node.ResourcesDuration = resourceDuration(wf, node, make(map[string]bool))
			wf.Status.Nodes.Set(nodeID, node)
		}
	}
}

func resourceDuration(wf *wfv1.Workflow, node wfv1.NodeStatus, visited map[string]bool) wfv1.ResourcesDuration {
	v := wfv1.ResourcesDuration{}
	for _, childID := range node.Children {
		// we do not want to visit the same node twice, as will (a) do 2x work and (b) make `v` incorrect
		if visited[childID] {
			continue
		}
		visited[childID] = true
		child, err := wf.Status.Nodes.Get(childID)
		if err != nil {
			log.Warnf("was unable to obtain node for %s", childID)
			continue
		}
		if child.Type == wfv1.NodeTypePod {
			v = v.Add(child.ResourcesDuration)
		}
		v = v.Add(resourceDuration(wf, *child, visited))
	}
	return v
}
