package main

import (
	"testing"

	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFrozenActionVersionCompatibility(t *testing.T) {
	ref := api.ResolvedReference{Kind: "ActionDefinition", Name: "action", UID: "identity", ResourceVersion: "7"}
	frozen := map[string]interface{}{"inputSchema": map[string]interface{}{"type": "object"}, "template": map[string]interface{}{"name": "call"}}
	digest := p.Digest(frozen)
	receipt := func(version string) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{"spec": map[string]interface{}{"definitionVersions": map[string]interface{}{"action/action": version}, "definitions": map[string]interface{}{"actions": map[string]interface{}{"action": frozen}}}}}
	}
	legacy := receipt("identity:7:" + digest)
	require.NoError(t, validateFrozenAction(ref, legacy))
	latest := receipt("identity:spec:" + digest)
	require.NoError(t, validateFrozenAction(ref, latest))
	ref.ResourceVersion = "status-only-update"
	require.NoError(t, validateFrozenAction(ref, latest))
	require.Error(t, validateFrozenAction(ref, legacy))
	ref.Digest = "different-content"
	require.Error(t, validateFrozenAction(ref, latest))
	ref.Digest = digest
	require.NoError(t, validateFrozenAction(ref, latest))
	ref.UID = "replacement"
	require.Error(t, validateFrozenAction(ref, latest))
	ref.UID = "identity"
	delete(latest.Object["spec"].(map[string]interface{}), "definitions")
	require.Error(t, validateFrozenAction(ref, latest))
	latest = receipt("identity:spec:" + "not-a-digest")
	require.Error(t, validateFrozenAction(ref, latest))
	// Frozen typed Argo templates may serialize zero-value fields absent in the
	// source. Receipt identities refer to the immutable source, not that encoding.
	latest = receipt("identity:spec:" + digest)
	require.NoError(t, unstructured.SetNestedField(latest.Object, map[string]interface{}{}, "spec", "definitions", "actions", "action", "template", "metadata"))
	require.NoError(t, validateFrozenAction(ref, latest))
}
