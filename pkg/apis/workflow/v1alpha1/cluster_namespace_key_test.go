package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterNamespaceKey(t *testing.T) {
	key, err := NewClusterNamespaceKey("clusterName", "namespace")
	assert.NoError(t, err)
	assert.Equal(t, key.String(), "clusterName.namespace")

	_, err = NewClusterNamespaceKey("clusterName", "")
	assert.Error(t, err)
	assert.Equal(t, err, fmt.Errorf(ErrIncompleteClusterNamespaceKey, "clusterName", ""))

	key, err = NewClusterNamespaceKey("cluster.Name", "namespace")
	assert.Error(t, err)
	assert.Equal(t, err, fmt.Errorf(ErrInvalidClusterNamespaceKey, key))
}
