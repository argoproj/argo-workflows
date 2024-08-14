package common

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestUnstructuredHasCompletedLabel(t *testing.T) {
	noLabel := &unstructured.Unstructured{}
	require.False(t, UnstructuredHasCompletedLabel(noLabel))

	label := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				LabelKeyCompleted: "true",
			},
		},
	}}
	require.True(t, UnstructuredHasCompletedLabel(label))

	falseLabel := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				LabelKeyCompleted: "false",
			},
		},
	}}
	require.False(t, UnstructuredHasCompletedLabel(falseLabel))

	unknownObject := "hello"
	require.False(t, UnstructuredHasCompletedLabel(unknownObject))
}
