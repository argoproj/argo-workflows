package progress

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/graph"
)

type Updator struct {
	podInformer cache.SharedIndexInformer
	wf          *wfv1.Workflow
	Updated     bool
}

func NewUpdator(podInformer cache.SharedIndexInformer, wf *wfv1.Workflow) *Updator {
	return &Updator{podInformer, wf, false}
}

var _ graph.Visitor = &Updator{}

func (u *Updator) Init() {
	u.wf.Status.Progress = "0/0"
}

func (u *Updator) Visit(nodeID string) {
	node := u.wf.Status.Nodes[nodeID]
	// unlike resource duration, progress can change
	progress := wfv1.Progress("")
	if node.IsLeaf() {
		if node.Type == wfv1.NodeTypePod {
			progress = u.podProgress(node, node.Progress)
		}
		// bit of a cheat, we kind of assume `0/1` is always set by the controller, not the pod
		// and that if it is fulfilled, it should be complete
		if node.Fulfilled() && (progress == "" || progress == "0/1") {
			progress = "1/1"
		} else if progress == "" {
			progress = "0/1"
		}
	} else {
		progress = "0/0"
		for _, childNodeID := range node.Children {
			// this will tolerate missing child (will be "") and therefore ignored
			v := u.wf.Status.Nodes[childNodeID].Progress
			if v.IsValid() {
				progress = progress.Add(v)
			}
		}
	}
	if progress.IsValid() && node.Progress != progress {
		node.Progress = progress
		u.wf.Status.Nodes[nodeID] = node
		u.Updated = true
	}
	if node.IsLeaf() {
		u.wf.Status.Progress = u.wf.Status.Progress.Add(node.Progress)
	}
}

func (u *Updator) podProgress(node wfv1.NodeStatus, progress wfv1.Progress) wfv1.Progress {
	// for pods, lets see what the annotation says pod can get deleted of course, so
	// can be empty and return "", even it previously had a value
	obj, _, _ := u.podInformer.GetStore().GetByKey(u.wf.Namespace + "/" + node.ID)
	if pod, ok := obj.(*apiv1.Pod); ok {
		if annotation, ok := pod.Annotations[common.AnnotationKeyProgress]; ok {
			v, ok := wfv1.ParseProgress(annotation)
			if ok {
				return v
			}
		}
	}
	return progress
}
