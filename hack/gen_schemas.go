package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

/*
	Generate JSON schemas good enough for the UI to validate against.

	We're a bit lazy with these, we do not scrub the assets, and we just bundle all
	the definitions, when we could cherry-pick the relevant ones.
*/
func genSchemas() {
	data, err := ioutil.ReadFile("api/openapi-spec/swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.Unmarshal(data, &swagger)
	if err != nil {
		panic(err)
	}
	for _, crd := range workflow.CRDs {
		name := "io.argoproj.workflow.v1alpha1." + crd.Kind
		filename := "ui/src/app/assets/schemas/" + crd.Kind + ".json"
		println(filename)
		schema := obj{
			"$id":         "http://workflows.argoproj.io/" + crd.Kind + ".json",
			"$schema":     "http://json-schema.org/draft-07/schema",
			"$ref":        "#/definitions/" + name,
			"definitions": swagger["definitions"],
		}
		data, err = json.MarshalIndent(schema, "", "  ")
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(filename, data, 0666)
		if err != nil {
			panic(err)
		}
	}
}
