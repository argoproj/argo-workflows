package controller

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/utils/env"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfretry "github.com/argoproj/argo-workflows/v3/workflow/util/retry"
)

// RetryTweak is a 2nd order function interface for tweaking the retry
type RetryTweak = func(retryStrategy wfv1.RetryStrategy, nodes wfv1.Nodes, pod *apiv1.Pod)

// FindRetryNode locates the closes retry node ancestor to nodeID
func FindRetryNode(nodes wfv1.Nodes, nodeID string) *wfv1.NodeStatus {
	boundaryID := nodes[nodeID].BoundaryID
	boundaryNode := nodes[boundaryID]
	for _, node := range nodes {
		if node.Type != wfv1.NodeTypeRetry {
			continue
		}
		if boundaryID == "" && node.HasChild(nodeID) {
			return &node
		} else if boundaryNode.TemplateName != "" && node.TemplateName == boundaryNode.TemplateName {
			return &node
		} else if boundaryNode.TemplateRef != nil && node.TemplateRef != nil && node.TemplateRef.Name == boundaryNode.TemplateRef.Name && node.TemplateRef.Template == boundaryNode.TemplateRef.Template {
			return &node
		}
	}
	return nil
}

// RetryOnDifferentHost append affinity with fail host to pod
func RetryOnDifferentHost(retryNodeName string) RetryTweak {
	return func(retryStrategy wfv1.RetryStrategy, nodes wfv1.Nodes, pod *apiv1.Pod) {
		if retryStrategy.Affinity == nil {
			return
		}
		hostNames := wfretry.GetFailHosts(nodes, retryNodeName)
		hostLabel := env.GetString("RETRY_HOST_NAME_LABEL_KEY", "kubernetes.io/hostname")
		if hostLabel != "" && len(hostNames) > 0 {
			pod.Spec.Affinity = wfretry.AddHostnamesToAffinity(hostLabel, hostNames, pod.Spec.Affinity)
		}
	}
}
