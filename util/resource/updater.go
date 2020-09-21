package resource

import (
	"strings"

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

func names(nodes wfv1.Nodes, children []string) []string {
	names := make([]string, len(children))
	for i, child := range children {
		names[i] = nodes[child].Name
	}
	return names
}

func (u *Updater) Visit(nodeID string) {
	nodes := u.wf.Status.Nodes
	node := nodes[nodeID]
	println(">", node.Name, node.Type, node.IsLeaf(), node.Phase, strings.Join(names(nodes, node.Children), ","), node.ResourcesDuration.String())
	if !node.IsLeaf() {
		if node.Fulfilled() {
			v := wfv1.ResourcesDuration{}
			for _, childID := range node.Children {
				// this will tolerate missing child (will be 0) and therefore ignored
				v = v.Add(nodes[childID].ResourcesDuration)
			}
			node.ResourcesDuration = v
			nodes[nodeID] = node
		}
	} else {
		u.wf.Status.ResourcesDuration = u.wf.Status.ResourcesDuration.Add(node.ResourcesDuration)
	}
	println("<", node.Name, node.Type, node.IsLeaf(), node.Phase, strings.Join(names(nodes, node.Children), ","), node.ResourcesDuration.String())
}
