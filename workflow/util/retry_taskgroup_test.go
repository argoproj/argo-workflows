package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestNativeRetryPreservesCompletedPredecessorTaskGroup(t *testing.T) {
	for _, retryChild := range []bool{false, true} {
		t.Run(map[bool]string{false: "retry downstream", true: "retry actual loop child"}[retryChild], func(t *testing.T) {
			wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf"}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowFailed, Nodes: wfv1.Nodes{}}}
			for _, n := range []wfv1.NodeStatus{
				{ID: "wf", Name: "wf", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeFailed, Children: []string{"group"}},
				{ID: "group", Name: "wf.keep", BoundaryID: "wf", Type: wfv1.NodeTypeTaskGroup, Phase: wfv1.NodeSucceeded, Children: []string{"left", "right"}},
				{ID: "left", Name: "wf.keep(0:left)", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, Children: []string{"next"}},
				{ID: "right", Name: "wf.keep(1:right)", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, Children: []string{"next"}},
				{ID: "next", Name: "wf.next", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeFailed},
			} {
				n.StartedAt = metav1.NewTime(time.Unix(10, 0))
				n.FinishedAt = metav1.NewTime(time.Unix(20, 0))
				wf.Status.Nodes[n.ID] = n
			}
			selector := ""
			if retryChild {
				selector = "id=left"
			}
			after, _, err := FormulateRetryWorkflow(logging.TestContext(t.Context()), wf, retryChild, selector, nil)
			require.NoError(t, err)
			group := after.Status.Nodes["group"]
			if retryChild {
				require.Equal(t, wfv1.NodeRunning, group.Phase)
				require.NotContains(t, after.Status.Nodes, "left")
			} else {
				group.OutboundNodes = nil // Native graph cleanup normalizes empty adjacency slices.
				require.Equal(t, wf.Status.Nodes["group"], group)
			}
		})
	}
}

func TestCheckedParameterRetryRetainsOnlyMatchingSiblingProof(t *testing.T) {
	for _, mode := range []string{"native", "checked", "wrong UID", "wrong version", "wrong fingerprint", "explicitly selected"} {
		t.Run(mode, func(t *testing.T) {
			wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "uid", ResourceVersion: "17"}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowFailed, Nodes: wfv1.Nodes{
				"wf":     {ID: "wf", Name: "wf", Type: wfv1.NodeTypeSteps, Phase: wfv1.NodeFailed, Children: []string{"group"}},
				"group":  {ID: "group", Name: "wf[0]", BoundaryID: "wf", Type: wfv1.NodeTypeStepGroup, Phase: wfv1.NodeFailed, Children: []string{"keep", "change"}},
				"keep":   {ID: "keep", Name: "wf[0].keep", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded},
				"change": {ID: "change", Name: "wf[0].change", BoundaryID: "wf", Type: wfv1.NodeTypePod, Phase: wfv1.NodeFailed},
			}}}
			ctx := logging.TestContext(t.Context())
			proof := p.NativeRetryProof{UID: "uid", ResourceVersion: "17", RetainedNodes: map[string]string{"keep": p.ResultFingerprint(wf.Status.Nodes["keep"])}}
			switch mode {
			case "wrong UID":
				proof.UID = "other"
			case "wrong version":
				proof.ResourceVersion = "18"
			case "wrong fingerprint":
				proof.RetainedNodes["keep"] = "incorrect"
			}
			if mode != "native" {
				ctx = p.WithRetryProof(ctx, proof)
			}
			selector := "id=change"
			if mode == "explicitly selected" {
				selector = "id!=wf,id!=group"
			}
			after, _, err := FormulateRetryWorkflow(ctx, wf, true, selector, []string{"value=new"})
			require.NoError(t, err)
			_, retained := after.Status.Nodes["keep"]
			require.Equal(t, mode == "checked", retained)
			if retained {
				require.Equal(t, p.ResultFingerprint(wf.Status.Nodes["keep"]), p.ResultFingerprint(after.Status.Nodes["keep"]))
			}
		})
	}
}
