package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

type crdType string

const (
	minimal crdType = "minimal"
	simple          = "simple"
	full            = "full"
)

/*
	Generate CRDs.

	controller-tools mandates you have good base types - but we have a number
	of problems I don't want to fix.
*/
func genCRDs() {
	for _, t := range []crdType{minimal, simple, full} {
		for _, crd := range workflow.CRDs {
			filename := "manifests/base/crds/" + strings.ToLower(string(t)) + "/" + crd.FullName + "-crd.yaml"

			println(filename)

			resource := obj{
				"apiVersion": "apiextensions.k8s.io/v1",
				"kind":       "CustomResourceDefinition",
				"metadata":   obj{"name": crd.FullName},
				"spec": obj{
					"conversion": obj{"strategy": "none"},
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
							"served":  true,
							"storage": true,
						},
					},
				},
			}

			if t != minimal {
				data, err := ioutil.ReadFile("api/openapi-spec/swagger.json")
				if err != nil {
					panic(err)
				}
				swagger := obj{}
				err = json.Unmarshal(data, &swagger)
				if err != nil {
					panic(err)
				}

				ssc := structuralSchemaContext{swagger: swagger, simple: t == simple}
				schema := ssc.structuralSchema(ssc.structuralSchemaByName("io.argoproj.workflow.v1alpha1." + crd.Kind))
				schema["required"] = []string{"metadata", "spec"}
				schema["properties"].(obj)["status"] = any
				resource["spec"].(obj)["versions"].(array)[0].(obj)["schema"] = obj{"openAPIV3Schema": schema}
			}

			data, err := yaml.Marshal(resource)
			if err != nil {
				panic(err)
			}
			err = ioutil.WriteFile(filename, data, 0666)
			if err != nil {
				panic(err)
			}
		}
	}
}
