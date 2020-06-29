package util

import "sigs.k8s.io/yaml"

func MustUnmarshallYAML(text string, v interface{}) {
	err := yaml.Unmarshal([]byte(text), v)
	if err != nil {
		panic(err)
	}
}
