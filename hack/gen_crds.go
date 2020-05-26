package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ghodss/yaml"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

/*
	Generate CRDs.

	controller-tools mandates you have good base types - but we have a number
	of problems I don't want to fix.


*/
func genCRDs() {
	data, err := ioutil.ReadFile("api/openapi-spec/swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.Unmarshal(data, &swagger)
	if err != nil {
		panic(err)
	}

	ctx := structuralSchemaContext{swagger: swagger, simple: true}

	for _, crd := range workflow.CRDs {
		filename := "manifests/base/crds/" + crd.FullName + "-crd.yaml"

		println(filename)

		schema := ctx.structuralSchema(ctx.structuralSchemaByName("io.argoproj.workflow.v1alpha1." + crd.Kind))
		schema["required"] = []string{"metadata", "spec"}
		schema["properties"].(obj)["status"] = any

		crd := obj{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata":   obj{"name": crd.FullName},
			"spec": obj{
				"conversion": obj{"strategy": "None"},
				"group":      workflow.Group,
				"names": obj{
					"kind":       crd.Kind,
					"listKind":   crd.Kind + "List",
					"plural":     crd.Plural,
					"shortNames": array{crd.ShortName},
					"singular":   crd.Singular,
				},
				"scope": crd.Scope,
				"versions": array{
					obj{
						"name":    "v1alpha1",
						"schema":  obj{"openAPIV3Schema": schema},
						"served":  true,
						"storage": true,
					},
				},
			},
		}
		data, err = yaml.Marshal(crd)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(filename, data, 0666)
		if err != nil {
			panic(err)
		}
	}
}
