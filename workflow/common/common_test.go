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

func TestGenSourceFilePath(t *testing.T) {
	testCases := []struct {
		name     string
		ctrName  string
		ctrType  string
		expected string
	}{
		{
			name:     "init container c1",
			ctrName:  "c1",
			ctrType:  "init",
			expected: "/argo/staging/init-c1-script",
		},
		{
			name:     "sidecar container c2",
			ctrName:  "c2",
			ctrType:  "sidecar",
			expected: "/argo/staging/sidecar-c2-script",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := GenSourceFilePath(tc.ctrType, tc.ctrName)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
