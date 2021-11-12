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

// PodNameVersion stores which type of pod names should be used.
// v1 represents the node id.
// v2 is the combination of a node id and template name.
type PodNameVersion string

const (
	// PodNameV1 is the v1 name that uses node ids for pod names
	PodNameV1 PodNameVersion = "v1"
	// PodNameV2 is the v2 name that uses node id combined with
	// the template name
	PodNameV2 PodNameVersion = "v2"
)

// String stringifies the pod name version
func (v PodNameVersion) String() string {
	return string(v)
}

// GetPodNameVersion returns the pod name version to be used
func GetPodNameVersion() PodNameVersion {
	switch os.Getenv("POD_NAMES") {
	case "v2":
		return PodNameV2
	case "v1":
		return PodNameV1
	default:
		return PodNameV1
	}
}

// PodName return a deterministic pod name
func PodName(workflowName, nodeName, templateName, nodeID string) string {
	if GetPodNameVersion() == PodNameV1 {
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
