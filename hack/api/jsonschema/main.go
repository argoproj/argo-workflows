package main

import (
	"encoding/json"
	"os"
)

func main() {
	swagger := obj{}
	{
		f, err := os.Open("api/openapi-spec/swagger.json")
		if err != nil {
			panic(err)
		}
		err = json.NewDecoder(f).Decode(&swagger)
		if err != nil {
			panic(err)
		}
	}
	{
		definitions := swagger["definitions"]
		for _, kind := range []string{
			"CronWorkflow",
			"ClusterWorkflowTemplate",
			"Workflow",
			"WorkflowEventBinding",
			"WorkflowTemplate",
		} {
			v := definitions.(obj)["io.argoproj.workflow.v1alpha1."+kind].(obj)
			v["x-kubernetes-group-version-kind"] = []map[string]string{
				{
					"group":   "argoproj.io",
					"kind":    kind,
					"version": "v1alpha1",
				},
			}
			props := v["properties"].(obj)
			props["apiVersion"].(obj)["const"] = "argoproj.io/v1alpha1"
			props["kind"].(obj)["const"] = kind
		}
		schema := obj{
			"$id":     "https://raw.githubusercontent.com/argoproj/argo-workflows/HEAD/api/jsonschema/schema.json",
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"type":    "object",
			"oneOf": []interface{}{
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.CronWorkflow"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.Workflow"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowEventBinding"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowTemplate"},
			},
			"definitions": definitions,
		}
		f, err := os.Create("api/jsonschema/schema.json")
		if err != nil {
			panic(err)
		}
		e := json.NewEncoder(f)
		e.SetIndent("", "  ")
		err = e.Encode(schema)
		if err != nil {
			panic(err)
		}
		err = f.Close()
		if err != nil {
			panic(err)
		}
	}
}
