package controller

import (
	"fmt"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
)

// TestProcessNodesWithRetries tests the processNodesWithRetries() method.
func TestProcessNodesWithRetries(t *testing.T) {
	controller := newController()
	assert.NotNil(t, controller)
	wf := unmarshalWF(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)

	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	var retryLimit int32
	retryLimit = 2
	retries.Limit = &retryLimit
	node.RetryStrategy = &retries
	woc.wf.Status.Nodes[nodeID] = *node

	retryNodes := woc.wf.Status.GetNodesWithRetries()
	assert.Equal(t, len(retryNodes), 1)
	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild, err := woc.getLastChildNode(node)
	assert.Nil(t, err)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.markNodePhase(childNode, wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.getNodeByName(nodeName)
	lastChild, err = woc.getLastChildNode(n)
	assert.Nil(t, err)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	err = woc.processNodeRetries(n)
	assert.Nil(t, err)
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	err = woc.processNodeRetries(n)
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)

	// Mark the parent node as running again and the lastChild as failed.
	woc.markNodePhase(n.Name, wfv1.NodeRunning)
	woc.markNodePhase(lastChild.Name, wfv1.NodeFailed)
	woc.processNodeRetries(n)
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Add a third node that has failed.
	childNode := "child-node-3"
	woc.markNodePhase(childNode, wfv1.NodeFailed)
	woc.addChildNode(nodeName, childNode)
	n = woc.getNodeByName(nodeName)
	err = woc.processNodeRetries(n)
	assert.Nil(t, err)
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeFailed)
}
