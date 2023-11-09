package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestUnstructuredHasCompletedLabel(t *testing.T) {
	noLabel := &unstructured.Unstructured{}
	assert.False(t, UnstructuredHasCompletedLabel(noLabel))

	label := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				LabelKeyCompleted: "true",
			},
		},
	}}
	assert.True(t, UnstructuredHasCompletedLabel(label))

	falseLabel := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				LabelKeyCompleted: "false",
			},
		},
	}}
	assert.False(t, UnstructuredHasCompletedLabel(falseLabel))

	unknownObject := "hello"
	assert.False(t, UnstructuredHasCompletedLabel(unknownObject))
}
