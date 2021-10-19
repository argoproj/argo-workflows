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
// v2 is the combination of a workflow name and template name.
type PodNameVersion string

const (
	// PodNameVersion1 is the v1 name that uses node ids for pod names
	PodNameVersion1 PodNameVersion = "v1"
	// PodNameVersion2 is the v2 name that uses workflow name combined with
	// the template name
	PodNameVersion2 PodNameVersion = "v2"
)

// String stringifies the pod name version
func (v PodNameVersion) String() string {
	return string(v)
}

// GetPodNameVersion returns the pod name version to be used
func GetPodNameVersion() PodNameVersion {
	switch os.Getenv("POD_NAMES") {
	case "v2":
		return PodNameVersion2
	case "v1":
		return PodNameVersion1
	default:
		return PodNameVersion1
	}
}

// PodName return a deterministic pod name
func PodName(workflowName, nodeName, templateName, nodeID string) string {
	if GetPodNameVersion() == PodNameVersion1 {
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
