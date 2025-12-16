package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetNestedField(t *testing.T) {
	o := &obj{
		"a": map[string]interface{}{
			"b": map[string]interface{}{},
		},
	}
	o.SetNestedField("newValue", "a", "b", "c")
	val := (*o)["a"].(map[string]interface{})["b"].(map[string]interface{})["c"]
	assert.Equal(t, "newValue", val)
}

func TestCopyNestedField(t *testing.T) {
	nested_o := &obj{
		"items": map[string]interface{}{
			"properties": map[string]interface{}{
				"name": "foo",
			},
		},
	}
	o := &obj{
		"steps": map[string]interface{}{
			"items": map[string]interface{}{
				"properties": map[string]interface{}{
					"steps": nested_o,
				},
			},
		},
	}
	o.CopyNestedField([]string{"steps", "items", "properties", "steps"}, []string{"steps", "items"})
	assert.Equal(t, &obj{"steps": map[string]interface{}{"items": nested_o}}, o)
}

func TestName(t *testing.T) {
	expectedName := "test-name"
	o := &obj{
		"metadata": map[string]interface{}{
			"name": expectedName,
		},
	}
	assert.Equal(t, expectedName, o.Name())
}

func TestOpenAPIV3Schema(t *testing.T) {
	expectedSchema := map[string]interface{}{"foo": "bar"}
	o := &obj{
		"spec": map[string]interface{}{
			"versions": []interface{}{
				map[string]interface{}{
					"schema": map[string]interface{}{
						"openAPIV3Schema": map[string]interface{}{
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
		"spec": map[string]interface{}{
			"schema": map[string]interface{}{
				"description": "desc",
				"properties": map[string]interface{}{
					"description": "Test",
					"a": map[string]interface{}{
						"description": "descA",
					},
					"b": map[string]interface{}{
						"description": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}
	o.RecursiveRemoveDescriptions("spec", "schema")
	assert.Equal(t, &obj{
		"spec": map[string]interface{}{
			"schema": map[string]interface{}{
				"description": "desc.\nAll nested field descriptions have been dropped due to Kubernetes size limitations.",
				"properties": map[string]interface{}{
					"a": map[string]interface{}{},
					"b": map[string]interface{}{
						"description": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}, o)
}
