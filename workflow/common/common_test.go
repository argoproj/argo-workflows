package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestUnstructuredHasCompletedLabel(t *testing.T) {
	noLabel := &unstructured.Unstructured{}
	assert.False(t, UnstructuredHasCompletedLabel(noLabel))

	label := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				LabelKeyCompleted: "true",
			},
		},
	}}
	assert.True(t, UnstructuredHasCompletedLabel(label))

	falseLabel := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				LabelKeyCompleted: "false",
			},
		},
	}}
	assert.False(t, UnstructuredHasCompletedLabel(falseLabel))

	unknownObject := "hello"
	assert.False(t, UnstructuredHasCompletedLabel(unknownObject))
}

func TestManifestDocCount(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		want     int
	}{
		{"single", "apiVersion: v1\nkind: ConfigMap", 1},
		{"leading-separator", "---\napiVersion: v1\nkind: ConfigMap", 1},
		{"trailing-separator", "apiVersion: v1\nkind: ConfigMap\n---\n", 1},
		{"multi", "kind: ConfigMap\n---\nkind: Secret", 2},
		{"multi-with-trailing", "kind: ConfigMap\n---\nkind: Secret\n---\n", 2},
		// the "..." document-end marker must count as a boundary; a naive split on "---" misses it
		{"multi-with-dots", "kind: ConfigMap\n...\nkind: Secret", 2},
		{"single-with-trailing-dots", "kind: ConfigMap\n...\n", 1},
		{"dots-then-separator", "kind: ConfigMap\n...\n---\nkind: Secret", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ManifestDocCount([]byte(tt.manifest)))
		})
	}
}
