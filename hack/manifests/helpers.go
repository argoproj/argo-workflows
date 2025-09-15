package main

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type obj map[string]interface{}

func (o *obj) RemoveNestedField(fields ...string) {
	unstructured.RemoveNestedField(*o, fields...)
}

func (o *obj) RecursiveRemoveDescriptions(fields ...string) {
	startField := nestedFieldNoCopy[map[string]interface{}](o, fields...)
	description := startField["description"].(string)
	startField["description"] = description + ".\nAll nested field descriptions have been dropped due to Kubernetes size limitations."

	var rec func(field *map[string]interface{})
	rec = func(field *map[string]interface{}) {
		if _, ok := (*field)["description"].(string); ok {
			delete(*field, "description")
		}
		for _, value := range *field {
			if nested, ok := value.(map[string]interface{}); ok {
				rec(&nested)
			}
		}
	}

	properties := startField["properties"].(map[string]interface{})
	rec(&properties)
}

func (o *obj) SetNestedField(value interface{}, fields ...string) {
	parentField := nestedFieldNoCopy[map[string]interface{}](o, fields[:len(fields)-1]...)
	parentField[fields[len(fields)-1]] = value
}

func (o *obj) CopyNestedField(sourceFields []string, targetFields []string) {
	value := nestedFieldNoCopy[any](o, sourceFields...)
	o.SetNestedField(value, targetFields...)
}

func (o *obj) Name() string {
	return nestedFieldNoCopy[string](o, "metadata", "name")
}

func (o *obj) OpenAPIV3Schema() obj {
	versions := nestedFieldNoCopy[[]interface{}](o, "spec", "versions")
	version := obj(versions[0].(map[string]interface{}))
	return nestedFieldNoCopy[map[string]interface{}](&version, "schema", "openAPIV3Schema", "properties")
}

func nestedFieldNoCopy[T any](o *obj, fields ...string) T {
	value, found, err := unstructured.NestedFieldNoCopy(*o, fields...)
	if !found {
		panic(fmt.Sprintf("failed to find field %v", fields))
	}
	if err != nil {
		panic(err.Error())
	}
	return value.(T)
}

func (o *obj) WriteYaml(filename string) {
	data, err := yaml.Marshal(o)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filename, data, 0o600)
	if err != nil {
		panic(err)
	}
}

func Read(filename string) []byte {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		panic(err)
	}
	return data
}

func ParseYaml(data []byte) *obj {
	crd := make(obj)
	err := yaml.Unmarshal(data, &crd)
	if err != nil {
		panic(err)
	}
	return &crd
}
