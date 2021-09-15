package util

import (
	"fmt"
	"hash/fnv"
	"os"
)

const (
	maxK8sResourceNameLength = 253
	k8sNamingHashLength      = 10
)

// PodName return a deterministic pod name
func PodName(workflowName, nodeName, templateName, nodeID string) string {
	if os.Getenv("POD_NAMES") == "v1" {
		return nodeID
	}

	if workflowName == nodeName {
		return workflowName
	}

	prefix := fmt.Sprintf("%s-%s", workflowName, templateName)
	prefix = ensurePodNamePrefixLength(prefix)

	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeName))
	return fmt.Sprintf("%s-%v", prefix, h.Sum32())
}

func ensurePodNamePrefixLength(prefix string) string {
	maxPrefixLength := maxK8sResourceNameLength - k8sNamingHashLength

	if len(prefix) > maxPrefixLength-1 {
		return prefix[0 : maxPrefixLength-1]
	}

	return prefix
}
