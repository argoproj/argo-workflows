package resource

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func UpdateResourceDurations(ctx context.Context, wf *wfv1.Workflow) {
	wf.Status.ResourcesDuration = wfv1.ResourcesDuration{}
	for nodeID, node := range wf.Status.Nodes {
		// pods are already calculated and so we do not need to compute them,
		// AND they are the only node that should contribute to the total
		if node.Type == wfv1.NodeTypePod {
			wf.Status.ResourcesDuration = wf.Status.ResourcesDuration.Add(node.ResourcesDuration)
		} else if node.Fulfilled() {
			// compute the sum of all children
			node.ResourcesDuration = resourceDuration(ctx, wf, node, make(map[string]bool))
			wf.Status.Nodes.Set(ctx, nodeID, node)
		}
	}
}

func resourceDuration(ctx context.Context, wf *wfv1.Workflow, node wfv1.NodeStatus, visited map[string]bool) wfv1.ResourcesDuration {
	v := wfv1.ResourcesDuration{}
	for _, childID := range node.Children {
		// we do not want to visit the same node twice, as will (a) do 2x work and (b) make `v` incorrect
		if visited[childID] {
			continue
		}
		visited[childID] = true
		child, err := wf.Status.Nodes.Get(childID)
		if err != nil {
			logging.RequireLoggerFromContext(ctx).WithField("childID", childID).Warn(ctx, "was unable to obtain node")
			continue
		}
		if child.Type == wfv1.NodeTypePod {
			v = v.Add(child.ResourcesDuration)
		}
		v = v.Add(resourceDuration(ctx, wf, *child, visited))
	}
	return v
}
