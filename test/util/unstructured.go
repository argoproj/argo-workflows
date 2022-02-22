package util

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func MustUnmarshalUnstructured(text string) *unstructured.Unstructured {
	v := &unstructured.Unstructured{}
	err := yaml.UnmarshalStrict([]byte(text), v)
	if err != nil {
		panic(err)
	}
	return v
}
