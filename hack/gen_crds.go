package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"
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

	for _, kind := range kinds {
		singular := strings.ToLower(kind)
		plural := singular + "s"
		group := "argoproj.io"
		name := plural + "." + group
		filename := "manifests/base/crds/" + plural + "-crd.yaml"

		println(filename)

		schema := structuralSchema(swagger, structuralSchemaByName(swagger, "io.argoproj.workflow.v1alpha1."+kind))
		schema["required"] = []string{"metadata", "spec"}
		schema["properties"].(obj)["status"] = any

		crd := obj{
			"apiVersion": "apiextensions.k8s.io/v1",
			"kind":       "CustomResourceDefinition",
			"metadata":   obj{"name": name},
			"spec": obj{
				"conversion": obj{"strategy": "None"},
				"group":      group,
				"names": obj{
					"kind":     kind,
					"listKind": kind + "List",
					"plural":   plural,
					"shortNames": array{map[string]string{
						"ClusterWorkflowTemplate": "cwftmpl",
						"CronWorkflow":            "cronwf",
						"Workflow":                "wf",
						"WorkflowTemplate":        "tmpl",
					}[kind]},
					"singular": singular,
				},
				"scope": map[string]string{
					"ClusterWorkflowTemplate": "Cluster",
					"CronWorkflow":            "Namespaced",
					"Workflow":                "Namespaced",
					"WorkflowTemplate":        "Namespaced",
				}[kind],
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
