package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func TestNestedDAGCompletesAfterDownstreamRetry(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: nested-retry
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: media
        template: media
      - name: publish
        depends: media.Succeeded
        template: publish
  - name: media
    dag:
      tasks:
      - name: loop
        template: noop
        withItems: [one]
  - name: noop
    container:
      image: busybox
      command: ["true"]
  - name: publish
    retryStrategy:
      limit: "0"
    container:
      image: busybox
      command: ["true"]
`)
	wf.Labels = map[string]string{}
	wf.Status.Phase = wfv1.WorkflowFailed
	wf.Status.Nodes = wfv1.Nodes{}
	now := metav1.Now()
	add := func(name, boundary, template string, typ wfv1.NodeType, phase wfv1.NodePhase, children ...string) {
		ids := make([]string, len(children))
		for i, c := range children {
			ids[i] = wf.NodeID(c)
		}
		boundaryID := ""
		if boundary != "" {
			boundaryID = wf.NodeID(boundary)
		}
		wf.Status.Nodes[wf.NodeID(name)] = wfv1.NodeStatus{ID: wf.NodeID(name), Name: name, Type: typ, Phase: phase, BoundaryID: boundaryID, TemplateName: template, TemplateScope: "local/", Children: ids, StartedAt: now, FinishedAt: now}
	}
	add(wf.Name, "", "main", wfv1.NodeTypeDAG, wfv1.NodeFailed, wf.Name+".media")
	add(wf.Name+".media", wf.Name, "media", wfv1.NodeTypeDAG, wfv1.NodeSucceeded, wf.Name+".media.loop")
	add(wf.Name+".media.loop", wf.Name+".media", "", wfv1.NodeTypeTaskGroup, wfv1.NodeSucceeded, wf.Name+".media.loop(0:one)")
	add(wf.Name+".media.loop(0:one)", wf.Name+".media", "noop", wfv1.NodeTypePod, wfv1.NodeSucceeded, wf.Name+".publish")
	add(wf.Name+".publish", wf.Name, "publish", wfv1.NodeTypeRetry, wfv1.NodeFailed, wf.Name+".publish(0)")
	add(wf.Name+".publish(0)", wf.Name, "publish", wfv1.NodeTypePod, wfv1.NodeFailed)
	retried, _, err := util.FormulateRetryWorkflow(logging.TestContext(t.Context()), wf, false, "", nil)
	require.NoError(t, err)
	// Simulate only the retried publish completing. The controller must converge
	// without rerunning or repairing the previously completed media work.
	publish := retried.Status.Nodes[wf.NodeID(wf.Name+".publish")]
	publish.Phase = wfv1.NodeSucceeded
	publish.FinishedAt = now
	retried.Status.Nodes[publish.ID] = publish
	woc := newWoc(logging.TestContext(t.Context()), *retried)
	woc.operate(logging.TestContext(t.Context()))
	require.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	require.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Nodes[wf.NodeID(wf.Name+".media")].Phase)
}
