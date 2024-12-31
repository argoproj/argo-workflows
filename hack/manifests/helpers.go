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

func (o *obj) CopyNestedField(sourceFields []string, targetFields []string) {
	value := nestedFieldNoCopy[any](o, sourceFields...)
	parentField := nestedFieldNoCopy[map[string]interface{}](o, targetFields[:len(targetFields)-1]...)
	parentField[targetFields[len(targetFields)-1]] = value
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
