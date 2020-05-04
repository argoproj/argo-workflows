package controller

import (
	jsonpatch "github.com/evanphx/json-patch"
	"gopkg.in/yaml.v2"
)

func newDiff(a, b interface{}) (string, error) {
	aData, err := yaml.Marshal(a)
	if err != nil {
		return "", err
	}
	bData, err := yaml.Marshal(b)
	if err != nil {
		return "", err
	}
	patchData, err := jsonpatch.CreateMergePatch(aData, bData)
	if err != nil {
		return "", err
	}
	return string(patchData), nil
}
