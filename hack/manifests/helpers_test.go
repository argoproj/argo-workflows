package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetNestedField(t *testing.T) {
	o := &obj{
		"a": map[string]any{
			"b": map[string]any{},
		},
	}
	o.SetNestedField("newValue", "a", "b", "c")
	val := (*o)["a"].(map[string]any)["b"].(map[string]any)["c"]
	assert.Equal(t, "newValue", val)
}

func TestCopyNestedField(t *testing.T) {
	nested_o := &obj{
		"items": map[string]any{
			"properties": map[string]any{
				"name": "foo",
			},
		},
	}
	o := &obj{
		"steps": map[string]any{
			"items": map[string]any{
				"properties": map[string]any{
					"steps": nested_o,
				},
			},
		},
	}
	o.CopyNestedField([]string{"steps", "items", "properties", "steps"}, []string{"steps", "items"})
	assert.Equal(t, &obj{"steps": map[string]any{"items": nested_o}}, o)
}

func TestName(t *testing.T) {
	expectedName := "test-name"
	o := &obj{
		"metadata": map[string]any{
			"name": expectedName,
		},
	}
	assert.Equal(t, expectedName, o.Name())
}

func TestOpenAPIV3Schema(t *testing.T) {
	expectedSchema := map[string]any{"foo": "bar"}
	o := &obj{
		"spec": map[string]any{
			"versions": []any{
				map[string]any{
					"schema": map[string]any{
						"openAPIV3Schema": map[string]any{
							"properties": expectedSchema,
						},
					},
				},
			},
		},
	}
	schema := o.OpenAPIV3Schema()
	assert.Equal(t, obj(expectedSchema), schema)
}

func TestWriteYamlAndRead(t *testing.T) {
	o := &obj{"foo": "bar"}
	tempFile := filepath.Join(t.TempDir(), "test.yaml")
	err := o.WriteYaml(tempFile)
	require.NoError(t, err)
	data := Read(tempFile)
	assert.Equal(t, o, ParseYaml(data))
}

func TestParseYaml(t *testing.T) {
	yamlData := []byte("foo: bar\nbaz: qux\n")
	o := ParseYaml(yamlData)
	assert.Equal(t, &obj{"foo": "bar", "baz": "qux"}, o)
}

func TestRecursiveRemoveDescriptions(t *testing.T) {
	o := &obj{
		"spec": map[string]any{
			"schema": map[string]any{
				"description": "desc",
				"properties": map[string]any{
					"description": "Test",
					"a": map[string]any{
						"description": "descA",
					},
					"b": map[string]any{
						"description": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
	o.RecursiveRemoveDescriptions("spec", "schema")
	assert.Equal(t, &obj{
		"spec": map[string]any{
			"schema": map[string]any{
				"description": "desc.\nAll nested field descriptions have been dropped due to Kubernetes size limitations.",
				"properties": map[string]any{
					"a": map[string]any{},
					"b": map[string]any{
						"description": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}, o)
}
