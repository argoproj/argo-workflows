package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestResourceKey(t *testing.T) {
	key, err := NewResourceKey("clusterName", "namespace", "resource", schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "Service",
	})
	assert.NoError(t, err)
	assert.Equal(t, key.String(), "clusterName/namespace/resource/Service.v1.apps")

	key = ResourceKey("clusterName/namespace/resource/Service")
	err = key.Validate()
	assert.Error(t, err)
	assert.Equal(t, err, fmt.Errorf(ErrInvalidGroupVersionResource, "Service"))

	input := "clusterName/name/space/resource/Service.v1.apps"
	key = ResourceKey(input)
	err = key.Validate()
	assert.Error(t, err)
	assert.Equal(t, err, fmt.Errorf(ErrInvalidResourceKey, input))
}
