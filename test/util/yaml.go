package util

import (
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

func MustUnmarshallYAML(text string, v interface{}) {
	err := yaml.UnmarshalStrict([]byte(text), v)
	if err != nil {
		log.Warnf("invalid YAML: %v", err)
		err = yaml.Unmarshal([]byte(text), v)
	}
	if err != nil {
		panic(err)
	}
}
