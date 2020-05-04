package controller

import (
	"github.com/sergi/go-diff/diffmatchpatch"
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
	dmp := diffmatchpatch.New()
	return dmp.DiffPrettyText(dmp.DiffMain(string(aData), string(bData), false)), nil
}
