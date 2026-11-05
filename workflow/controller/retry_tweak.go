package controller

import (
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/utils/env"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfretry "github.com/argoproj/argo-workflows/v4/workflow/util/retry"
)

// RetryTweak is a 2nd order function interface for tweaking the retry
type RetryTweak = func(retryStrategy wfv1.RetryStrategy, nodes wfv1.Nodes, pod *apiv1.Pod)

// FindRetryNode locates the closest retry node ancestor to nodeID
func FindRetryNode(nodes wfv1.Nodes, nodeID string) *wfv1.NodeStatus {
	if parentNode := nodes.FindRetryNodeByChild(nodeID); parentNode != nil {
		return parentNode
	}
	boundaryID := nodes[nodeID].BoundaryID
	if parentNode := nodes.FindRetryNodeByChild(boundaryID); parentNode != nil {
		return parentNode
	}
	return nil
}

// FindRetryNodeByName finds the same node as FindRetryNode without scanning
// every node. A retry node names each attempt "<retry node name>(<attempt>)"
// (see handleRetries), so the only node that can be a node's retry parent is
// the one named by dropping that suffix, found by name in constant time.
// FindRetryNode visits every node of the workflow, which a controller
// creating thousands of pods in a large workflow pays once per pod.
func FindRetryNodeByName(wf *wfv1.Workflow, nodeID string) *wfv1.NodeStatus {
	if parentNode := retryParentByName(wf, nodeID); parentNode != nil {
		return parentNode
	}
	node, ok := wf.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return retryParentByName(wf, node.BoundaryID)
}

// retryParentByName returns the retry node that has childID as an attempt, if
// any.
func retryParentByName(wf *wfv1.Workflow, childID string) *wfv1.NodeStatus {
	child, ok := wf.Status.Nodes[childID]
	if !ok {
		return nil
	}
	parentName, ok := retryParentName(child.Name)
	if !ok {
		return nil
	}
	parentNode, err := wf.GetNodeByName(parentName)
	if err != nil || parentNode.Type != wfv1.NodeTypeRetry || !parentNode.HasChild(childID) {
		return nil
	}
	return parentNode
}

// retryParentName strips a retry attempt suffix, "(<digits>)", from a node
// name, reporting whether the name had one.
func retryParentName(name string) (string, bool) {
	if !strings.HasSuffix(name, ")") {
		return "", false
	}
	open := strings.LastIndex(name, "(")
	if open < 0 {
		return "", false
	}
	attempt := name[open+1 : len(name)-1]
	if attempt == "" || strings.Trim(attempt, "0123456789") != "" {
		return "", false
	}
	return name[:open], true
}

// RetryOnDifferentHost append affinity with fail host to pod
func RetryOnDifferentHost(retryNodeName string) RetryTweak {
	return func(retryStrategy wfv1.RetryStrategy, nodes wfv1.Nodes, pod *apiv1.Pod) {
		if retryStrategy.Affinity == nil {
			return
		}
		if retryStrategy.Affinity.NodeAntiAffinity != nil {
			hostNames := wfretry.GetFailHosts(nodes, retryNodeName)
			hostLabel := env.GetString("RETRY_HOST_NAME_LABEL_KEY", "kubernetes.io/hostname")
			if hostLabel != "" && len(hostNames) > 0 {
				pod.Spec.Affinity = wfretry.AddHostnamesToAffinity(hostLabel, hostNames, pod.Spec.Affinity)
			}
		}
	}
}
