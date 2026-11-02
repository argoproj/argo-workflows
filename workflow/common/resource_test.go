package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func TestDeterministicResourceNamePreservesIdentityForLongPrefixes(t *testing.T) {
	manifest := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  generateName: " + strings.Repeat("a", 80) + "\n"
	name := func(nodeID string) string {
		var obj unstructured.Unstructured
		require.NoError(t, yaml.Unmarshal([]byte(ReplaceGenerateNameWithDeterministic(manifest, nodeID)), &obj))
		return obj.GetName()
	}
	assert.Len(t, name("first"), 63)
	assert.NotEqual(t, name("first"), name("second"))
}

func TestResourceRecoveryGetFlagsPreservesClientConfiguration(t *testing.T) {
	flags, err := ResourceRecoveryGetFlags([]string{
		"-ntarget", "--context", "target-cluster", "--kubeconfig=/tmp/test-kubeconfig",
		"--as-group=first", "--as-group=second", "--request-timeout=30s",
		"--validate=false", "--field-manager=resource-test", "--save-config",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"--as-group=first", "--as-group=second", "--context=target-cluster",
		"--kubeconfig=/tmp/test-kubeconfig", "--namespace=target", "--request-timeout=30s",
	}, flags)
}
