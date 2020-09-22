package resource

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/graph"
)

type Updater struct {
	wf *wfv1.Workflow
}

var _ graph.Visitor = &Updater{}

func NewUpdater(wf *wfv1.Workflow) *Updater {
	return &Updater{wf}
}

func (u *Updater) Init() {
	u.wf.Status.ResourcesDuration = wfv1.ResourcesDuration{}
}

func (u *Updater) Visit(nodeID string) {
	nodes := u.wf.Status.Nodes
	node := nodes[nodeID]
	// pods are already calculated and so we do not need to compute them,
	// AND they are the only node that should contribute to the total
	if node.Type == wfv1.NodeTypePod {
		u.wf.Status.ResourcesDuration = u.wf.Status.ResourcesDuration.Add(node.ResourcesDuration)
	} else if node.Fulfilled() {
		// compute the sum of all children
		node.ResourcesDuration = u.resourceDuration(node, make(map[string]bool))
		nodes[nodeID] = node
	}
}

func (u *Updater) resourceDuration(node wfv1.NodeStatus, visited map[string]bool) wfv1.ResourcesDuration {
	v := wfv1.ResourcesDuration{}
	for _, childID := range node.Children {
		// we do not want to visit the same node twice, as will (a) do 2x work and (b) make `v` incorrect
		if visited[childID] {
			continue
		}
		visited[childID] = true
		child := u.wf.Status.Nodes[childID]
		if child.Type == wfv1.NodeTypePod {
			v = v.Add(child.ResourcesDuration)
		}
		v = v.Add(u.resourceDuration(child, visited))
	}
	return v
}
