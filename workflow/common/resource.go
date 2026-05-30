package common

import (
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// ReplaceGenerateNameWithDeterministic replaces generateName with a deterministic name
// derived from the node ID, enabling idempotent resource creation on executor pod restart.
// Uses the same fnv32a hashing pattern as GeneratePodName and Workflow.NodeID.
func ReplaceGenerateNameWithDeterministic(manifest, nodeID string) string {
	obj := unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(manifest), &obj); err != nil {
		return manifest
	}
	if obj.GetName() != "" || obj.GetGenerateName() == "" {
		return manifest
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeID))
	name := fmt.Sprintf("%s%08x", obj.GetGenerateName(), h.Sum32())
	if len(name) > 63 {
		name = name[:63]
	}
	obj.SetName(name)
	obj.SetGenerateName("")
	bytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return manifest
	}
	return string(bytes)
}
