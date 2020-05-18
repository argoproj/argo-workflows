package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

func genSchemaAssets() {
	for _, kind := range []string{"clusterworkflowtemplates", "cronworkflows", "workflows", "workflowtemplates"} {
		data, err := ioutil.ReadFile("manifests/base/crds/" + kind + ".argoproj.io-crd.yaml")
		if err != nil {
			panic(err)
		}
		crd := map[string]interface{}{}
		err = yaml.Unmarshal(data, &crd)
		if err != nil {
			panic(err)
		}
		openAPIV3Schema := crd["spec"].(map[string]interface{})["validation"].(map[string]interface{})["openAPIV3Schema"]
		data, err = json.MarshalIndent(openAPIV3Schema, "", "  ")
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile("ui/src/app/assets/schemas/"+kind+".json", data, 0666)
		if err != nil {
			panic(err)
		}
	}
}
