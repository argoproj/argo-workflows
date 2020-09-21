package resource

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/graph"
)

type Updator struct {
	wf      *wfv1.Workflow
	Updated bool
}

var _ graph.Visitor = &Updator{}

func NewUpdator(wf *wfv1.Workflow) *Updator {
	return &Updator{wf, false}
}

func (u *Updator) Init() {
	u.wf.Status.ResourcesDuration = wfv1.ResourcesDuration{}
}

func (u *Updator) Visit(nodeID string) {
	nodes := u.wf.Status.Nodes
	node := nodes[nodeID]
	// leaf nodes will have been computed, we only need to update those that have yet to be calculated
	if len(node.Children) > 0 && node.Fulfilled() && node.ResourcesDuration.IsZero() {
		v := wfv1.ResourcesDuration{}
		for _, childID := range node.Children {
			// this will tolerate missing child (will be 0) and therefore ignored
			v = v.Add(nodes[childID].ResourcesDuration)
		}
		node.ResourcesDuration = v
		nodes[nodeID] = node
		u.Updated = true
	}
	if node.IsLeaf() {
		u.wf.Status.ResourcesDuration = u.wf.Status.ResourcesDuration.Add(node.ResourcesDuration)
	}
}
