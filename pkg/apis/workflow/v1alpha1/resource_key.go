package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// ErrInvalidResourceKey gets kicked when a given ResourceKey doesn't follow
	// the clusterName/namespace/resourceName/resource.version.group convention
	ErrInvalidResourceKey string = "invalid resource key: resource key does not follow clusterName/namespace/resourceName/resource.version.group convention: %s"
	// ErrInvalidGroupVersionResource is thrown when the ending of a ResourceKey
	// doesn't conform to "resource.group.com" or "resource.version.group.com"
	ErrInvalidGroupVersionResource string = "invalid group version resource: cannot be parsed into either a group version resource or a group resource: %s"
)

// ResourceKey is a controller-level unique key for a resource
type ResourceKey string

// NewResourceKey creates a new resource key
func NewResourceKey(clusterName ClusterName, namespace, name string, gvr schema.GroupVersionResource) (ResourceKey, error) {
	key := ResourceKey(fmt.Sprintf("%s/%s/%s/%s.%s.%s", clusterName, namespace, name, gvr.Resource, gvr.Version, gvr.Group))
	return key, key.Validate()
}

// String stringifies a ResourceKey
func (key ResourceKey) String() string {
	return string(key)
}

// Validate determines whether a ResourceKey follows the
// clusterName/namespace/resourceName/resource.version.group convention
func (key *ResourceKey) Validate() error {
	parts := strings.Split(key.String(), "/")
	if len(parts) != 4 {
		return fmt.Errorf(ErrInvalidResourceKey, key.String())
	}

	x, _ := schema.ParseResourceArg(parts[3])
	if x.Empty() {
		return fmt.Errorf(ErrInvalidGroupVersionResource, parts[3])
	}

	return nil
}

// Split takes a ResourceKey and returns the component cluster name, namespace,
// resourceName, and GroupVersionResource
func (key *ResourceKey) Split() (ClusterName, string, string, schema.GroupVersionResource, error) {
	if err := key.Validate(); err != nil {
		return "", "", "", schema.GroupVersionResource{}, err
	}

	parts := strings.Split(key.String(), "/")
	x, _ := schema.ParseResourceArg(parts[3])

	return ClusterName(parts[0]), parts[1], parts[2], *x, nil
}
